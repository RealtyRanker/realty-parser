FROM --platform=linux/amd64 ghcr.io/userver-framework/ubuntu-24.04-userver

ENV CURRENT=/home/andrew-misnik/realty-parser
ENV LOGS_PATH=$CURRENT/logs

RUN rm -rf $LOGS_PATH
RUN pkill -9 userver || true
RUN mkdir -p configs
COPY configs/static_config.yaml conf/static_config.yaml
RUN mkdir -p bin
COPY build_release/realty-parser bin/

RUN apt-get update && apt-get install -y wget bzip2 build-essential
RUN apt-get install -y wget cmake build-essential
RUN apt-get install -y libgumbo-dev

RUN wget http://downloads.sourceforge.net/project/boost/boost/1.74.0/boost_1_74_0.tar.gz
RUN tar -xzf boost_1_74_0.tar.gz
RUN cd boost_1_74_0/ && ./bootstrap.sh --prefix=/usr/local
RUN cd boost_1_74_0/ && ./b2 install
RUN rm -rf /var/lib/apt/lists/* boost_1_74_0* 

CMD ["/bin/realty-parser", "-c", "./conf/static_config.yaml"]