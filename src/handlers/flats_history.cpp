#include "flats_history.hpp"

#include <fmt/format.h>

#include <userver/clients/dns/component.hpp>
#include <userver/components/component.hpp>
#include <userver/formats/json.hpp>
#include <userver/server/handlers/http_handler_base.hpp>
#include <userver/storages/postgres/cluster.hpp>
#include <userver/storages/postgres/component.hpp>
#include <userver/utils/assert.hpp>

namespace realty-parser {

namespace {

class FlatsHistoryHandler final : public userver::server::handlers::HttpHandlerBase {
  public:
    static constexpr std::string_view kName = "flats-history-handler";
  FlatsHistoryHandler(const userver::components::ComponentConfig& config,
        const userver::components::ComponentContext& component_context)
      : HttpHandlerBase(config, component_context),
        pg_cluster_(
            component_context
                .FindComponent<userver::components::Postgres>("postgres-db")
                .GetCluster()) {}

  std::string HandleRequestThrow(
      const userver::server::http::HttpRequest& request,
      userver::server::request::RequestContext&) const override {
    auto result = pg_cluster_->Execute(
        userver::storages::postgres::ClusterHostType::kMaster,
        "SELECT * FROM flats_history");
    
    userver::formats::json::ValueBuilder json_vb = {userver::formats::common::Type::kArray};
    for (auto& row : result) {
      userver::formats::json::ValueBuilder rowJson;
      std::set<std::string> doubleFields = {"underground_score", "total_area"};
      std::set<std::string> integerFields = {"price", "flat_score", "underground_place", "last_updated_timestamp"};
      for (auto& field : row) {
        if (doubleFields.find(std::string{field.Name()}) != doubleFields.end()) {
          rowJson[std::string{field.Name()}] = row[std::string{field.Name()}].As<double>();
        } else if (integerFields.find(std::string{field.Name()}) != integerFields.end()) {
          rowJson[std::string{field.Name()}] = row[std::string{field.Name()}].As<int>();
        } else {
          rowJson[std::string{field.Name()}] = row[std::string{field.Name()}].As<std::string>();
        }
      }
      json_vb.PushBack(rowJson.ExtractValue());
    }

    request.GetHttpResponse().SetContentType("application/json");
    request.GetHttpResponse().SetHeader(std::string("Access-Control-Allow-Origin"), std::string("*"));
    return userver::formats::json::ToString(json_vb.ExtractValue());
  }

  userver::storages::postgres::ClusterPtr pg_cluster_;
};

}  // namespace

void AppendFlatsHistoryHandler(userver::components::ComponentList& component_list) {
  component_list.Append<FlatsHistoryHandler>();
}

}  // namespace realty-parser
