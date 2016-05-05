FROM    laincloud/centos-lain

RUN     mkdir -p $GOPATH/src/github.com/laincloud/

ADD     . $GOPATH/src/github.com/laincloud/lainlet

RUN     cd $GOPATH/src/github.com/laincloud/lainlet && go build -v -a -tags netgo -installsuffix netgo -o lainlet

RUN     mv $GOPATH/src/github.com/laincloud/lainlet/lainlet /usr/bin/

