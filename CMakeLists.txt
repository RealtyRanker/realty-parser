cmake_minimum_required(VERSION 3.12)
project(realty-parser CXX)


# Adding userver dependency
find_package(userver COMPONENTS core postgresql s3api QUIET)
if(NOT userver_FOUND)  # Fallback to subdirectory usage
    # Enable userver libraries that are needed in this project
    set(USERVER_FEATURE_POSTGRESQL ON CACHE BOOL "" FORCE)
    set(USERVER_FEATURE_S3API ON CACHE BOOL "" FORCE)

    # Compatibility mode: some systems don't support these features
    set(USERVER_FEATURE_CRYPTOPP_BLAKE2 OFF CACHE BOOL "" FORCE)
    set(USERVER_FEATURE_GRPC_CHANNELZ OFF CACHE BOOL "" FORCE)
    set(USERVER_FEATURE_REDIS_HI_MALLOC ON CACHE BOOL "" FORCE)

    if (EXISTS third_party/userver)
        message(STATUS "Using userver framework from third_party/userver")
        add_subdirectory(third_party/userver)
    else()
        message(FATAL_ERROR "Either install the userver or provide a path to it")
    endif()
endif()

userver_setup_environment()

find_package(PkgConfig REQUIRED)
pkg_check_modules(Gumbo REQUIRED gumbo)
find_package(Threads REQUIRED)
find_package(OpenSSL REQUIRED)
find_package(Boost COMPONENTS system REQUIRED)
find_package(CURL)

add_subdirectory(third_party/tgbot-cpp)
include_directories(third_party/tgbot-cpp/include)


# Common sources
add_library(${PROJECT_NAME}_objs OBJECT
    src/backgrounds/latest_flats_listener.cpp
    src/backgrounds/latest_flats_listener.hpp

    src/components/s3api_client.cpp
    src/components/s3api_client.hpp
    src/components/telegram_bot.cpp
    src/components/telegram_bot.hpp

    src/handlers/flats_history.cpp
    src/handlers/flats_history.hpp
    src/handlers/hello.cpp
    src/handlers/hello.hpp
)
target_link_libraries(${PROJECT_NAME}_objs PUBLIC userver::postgresql userver::s3api)


# The Service
add_executable(${PROJECT_NAME} src/main.cpp)
target_link_libraries(${PROJECT_NAME} PRIVATE ${PROJECT_NAME}_objs)

target_link_libraries(${PROJECT_NAME}_objs PUBLIC ${Gumbo_LIBRARIES})
target_link_libraries(${PROJECT_NAME}_objs PUBLIC TgBot)

# Install
include(GNUInstallDirs)

if(DEFINED ENV{PREFIX})
  message(STATUS "Set install prefix: $ENV{PREFIX}")
  file(TO_CMAKE_PATH "$ENV{PREFIX}" PREFIX_PATH)
  set(CMAKE_INSTALL_PREFIX ${PREFIX_PATH})
endif()

file(GLOB CONFIGS_FILES ${CMAKE_CURRENT_SOURCE_DIR}/configs/*.yaml ${CMAKE_CURRENT_SOURCE_DIR}/configs/*.json)

install(TARGETS ${PROJECT_NAME} DESTINATION ${CMAKE_INSTALL_BINDIR} COMPONENT ${PROJECT_NAME})
install(FILES ${CONFIGS_FILES} DESTINATION ${CMAKE_INSTALL_SYSCONFDIR}/${PROJECT_NAME} COMPONENT ${PROJECT_NAME})
