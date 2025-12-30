# Realty Parser

Service for parsing realty info and storing it in various ways.

rm -rf build_release
cmake -B build_release \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_EXPORT_COMPILE_COMMANDS=ON \
    -DBOOST_ROOT=/opt/homebrew/opt/boost \
    -DUSERVER_FEATURE_PATCH_LIBPQ=OFF