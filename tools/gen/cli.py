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
//CODE GENERATION ${name} START
func (cli *Client) ${name}Get(${param_str}) (*pb.${name}Reply, error) {
	req := &pb.${name}Request{${req_init}}
	ctx, cancel := context.WithTimeout(context.Background(), cli.timeout)
	defer cancel()
	rpl, err := cli.${client}.Get(ctx, req)
	return rpl, err
}

type ${name}Watcher struct {
	err    error
	ch     chan *pb.${name}Reply
	sync.RWMutex
}

func (wch *${name}Watcher) Next() (*pb.${name}Reply, error) {
	rpl := <- wch.ch
	wch.RLock()
	defer wch.RUnlock()
	err := wch.err
	return rpl, err
}
func (cli *Client) ${name}Watch(${param_str}) (*${name}Watcher, error) {
	req := &pb.${name}Request{${req_init}}
	stream, err := cli.${client}.Watch(context.Background(), req)
	if err != nil {
		return nil, err
	}
	wch := &${name}Watcher{
		ch: make(chan *pb.${name}Reply),
	}
	go func() {
		defer close(wch.ch)
		for {
			in, err := stream.Recv()
			if err != nil {
				wch.Lock()
				wch.err = err
				wch.Unlock()
				return
			}
			wch.ch <- in
		}
	}()
	return wch, nil
}
//CODE GENERATION ${name} END
"""


def firstCharLower(ss):
    return ss[0].lower() + ss[1:]


def gen_client():
    name = sys.argv[1]
    param = None
    if len(sys.argv) == 3:
        param = sys.argv[2]
    param_str = ""
    req_init = ""
    if param is not None:
        param_str = "%s string" % param
        req_init = "%s: %s" % (param.capitalize(), param)
    fname = os.getenv("GOFILE")
    print("generate client for ", name)
    with open(fname, "r+") as fp:
        content = fp.read()
        d = {
            "name": name,
            "param_str": param_str,
            "req_init": req_init,
            "client": firstCharLower(name) + "Client",
        }
        regex = "//CODE GENERATION {name} START.*//CODE GENERATION {name} END".format(name=name)
        code = Template(code_tpl).substitute(d)
        code = code.strip("\n")
        if re.search(regex, content, re.DOTALL):
            content = re.sub(regex, code, content, flags=re.DOTALL)
        else:
            content += ("\n\n" + code)
        fp.seek(0)
        fp.write(content)
        fp.truncate(len(content))


gen_client()
