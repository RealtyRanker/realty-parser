#include "hello.hpp"
#include "../components/statistics.hpp"
#include <iostream>

#include <fmt/format.h>

#include <userver/clients/dns/component.hpp>
#include <userver/components/component.hpp>
#include <userver/server/handlers/http_handler_base.hpp>
#include <userver/storages/postgres/cluster.hpp>
#include <userver/storages/postgres/component.hpp>
#include <userver/utils/assert.hpp>

namespace realty-parser {

namespace {

class Hello final : public userver::server::handlers::HttpHandlerBase {
 public:
  static constexpr std::string_view kName = "hello-handler";

  Hello(const userver::components::ComponentConfig& config,
        const userver::components::ComponentContext& context)
      : HttpHandlerBase(config, context),
        pg_cluster_(
            context
                .FindComponent<userver::components::Postgres>("postgres-db")
                .GetCluster()),
        stats(realty-parser::components::Statistics(config, context)) {}

  std::string HandleRequestThrow(
      const userver::server::http::HttpRequest& request,
      userver::server::request::RequestContext&) const override {
    const auto& name = request.GetArg("name");
    LOG_INFO() <<"NAME = " << name;

    auto user_type = UserType::kFirstTime;
    if (!name.empty()) {
      auto result = pg_cluster_->Execute(
          userver::storages::postgres::ClusterHostType::kMaster,
          "INSERT INTO public.users(name, count) VALUES($1, 1) "
          "ON CONFLICT (name) "
          "DO UPDATE SET count = users.count + 1 "
          "RETURNING users.count",
          name);
      stats.AddMetric("my_metric", result.AsSingleRow<int>());
      if (result.AsSingleRow<int>() > 1) {
        user_type = UserType::kKnown;
      }
    }
    request.GetHttpResponse().SetContentType("application/json");
    request.GetHttpResponse().SetHeader(std::string("Access-Control-Allow-Origin"), std::string("*"));
    return realty-parser::SayHelloTo(name, user_type);
  }

  userver::storages::postgres::ClusterPtr pg_cluster_;
  realty-parser::components::Statistics stats;
};

}  // namespace

std::string SayHelloTo(std::string_view name, UserType type) {
  if (name.empty()) {
    name = "unknown user";
  }

  switch (type) {
    case UserType::kFirstTime:
      return fmt::format("Hello, {}!\n", name);
    case UserType::kKnown:
      return fmt::format("Hi again, {}!\n", name);
  }

  return "";
}

void AppendHelloHandler(userver::components::ComponentList& component_list) {
  component_list.Append<Hello>();
  component_list.Append<userver::components::Postgres>("postgres-db");
}

}  // namespace realty-parser
