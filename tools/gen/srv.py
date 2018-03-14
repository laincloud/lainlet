#!/usr/bin/env python3
# -*- coding:utf-8 -*-

"""

"""
from __future__ import print_function, division, absolute_import
import os
import sys
import re
from string import Template

code_tpl = """
//CODE GENERATION ${index} START
func (ed *${name}Endpoint) Get(ctx context.Context, in *pb.${name}Request) (*pb.${name}Reply, error) {
	key, err := ed.getKey(in, ctx)
	if err != nil {
		return nil, err
	}
	data, err := ed.wch.Get(key)
	if err != nil {
		return nil, err
	}
	obj, _, err := ed.make(key, data)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (ed *${name}Endpoint) Watch(in *pb.${name}Request, stream pb.${name}_WatchServer) error {
 	ctx := stream.Context()
	key, err := ed.getKey(in, ctx)
	if err != nil {
		return err
	}

	// send the initial data
	data, err := ed.wch.Get(key)
	if err != nil {
		return err
	}
	obj, _, err := ed.make(key, data)
	if err != nil {
		return err
	}
	if err := stream.Send(obj); err != nil {
		return err
	}

	// start watching
	ch, err := ed.wch.Watch(key, ctx)
	if err != nil {
		return fmt.Errorf("Fail to watch %s, %s", key, err.Error())
	}
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			// log.Infof("Get a %s event, id=%d action=%s ", api.WatcherName(), event.ID, event.Action.String())
			if event.Action == store.ERROR {
				err := fmt.Errorf("got an error from store, ID: %v, Data: %v", event.ID, event.Data)
				return err
			}
			obj, changed, err := ed.make(key, event.Data)
			if err != nil {
				return err
			}
			if !changed {
				continue
			}
			if err := stream.Send(obj); err != nil {
				return err
			}
		case  <-ctx.Done():
		    return nil
		}
	}
}

//CODE GENERATION ${index} END
"""


def gen_get_watch():
    index = int(sys.argv[1])
    name = sys.argv[2]

    fname = os.getenv("GOFILE")
    print(fname)
    with open(fname, "r+") as fp:
        content = fp.read()
        d = {
            "name": name,
            "index": index,
        }
        regex = "//CODE GENERATION {index} START.*//CODE GENERATION {index} END".format(index=index)
        code = Template(code_tpl).substitute(d)
        code = code.strip("\n")
        if re.search(regex, content, re.DOTALL):
            content = re.sub(regex, code, content, flags=re.DOTALL)
        else:
            content += ("\n\n" + code)
        fp.seek(0)
        fp.write(content)
        fp.truncate(len(content))


gen_get_watch()
