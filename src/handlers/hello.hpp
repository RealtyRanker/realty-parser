#pragma once

#include <string_view>

#include <userver/components/component_list.hpp>

namespace realty-parser {

enum class UserType { kFirstTime, kKnown };
std::string SayHelloTo(std::string_view name, UserType type);

void AppendHelloHandler(userver::components::ComponentList& component_list);

}  // namespace realty-parser
