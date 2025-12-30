

brew install make
brew install cmake --fix-missing
brew install build-essential --fix-missing

brew install python3-dev
brew install python3-venv
brew install libpq-dev

brew install libyaml-cpp-dev
brew install libzstd-dev
brew install libnghttp2-dev
brew install libev-dev
brew install libldap2-dev
brew install postgresql-server-dev-all
brew install libkrb5-dev
brew install libjemalloc-dev
brew install adduser libfontconfig1 musl
brew install libpugixml-dev
brew install libgumbo-dev


CURRENT=$(pwd)
export LOGS_PATH=$CURRENT/logs

cd $CURRENT && wget http://downloads.sourceforge.net/project/boost/boost/1.74.0/boost_1_74_0.tar.gz
cd $CURRENT && tar -xzf boost_1_74_0.tar.gz && cd boost_1_74_0 && ./bootstrap.sh && sudo ./b2 install
cd $CURRENT && mkdir third_party
cd $CURRENT && cd ./third_party && git clone --depth 1 https://github.com/userver-framework/userver.git
cd $CURRENT && pwd && sudo dpkg -i grafana-enterprise_11.1.0_amd64.deb

