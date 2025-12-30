#include "latest_flats_listener.hpp"
#include "../components/s3api_client.hpp"
#include "../components/telegram_bot.hpp"

#include "../utils/filter_utils.hpp"
#include "../utils/flat_info.hpp"
#include "../utils/request_utils.hpp"
#include "../utils/telegram_mailing_utils.hpp"

#include <gumbo.h>

#include <userver/components/component_base.hpp>
#include <userver/components/component_context.hpp>
#include <userver/utils/periodic_task.hpp>
#include <userver/yaml_config/merge_schemas.hpp>

#include <userver/clients/http/client.hpp>
#include <userver/clients/http/component.hpp>
#include <userver/clients/http/request.hpp>
#include <userver/s3api/clients/s3api.hpp>

void findLinkAreaTags(GumboNode* node, std::vector<GumboNode*>& result) {
  if (node->type != GUMBO_NODE_ELEMENT) {
    return;
  }

  GumboAttribute* attribute;
  if (node->v.element.tag == GUMBO_TAG_DIV &&
      (attribute = gumbo_get_attribute(&node->v.element.attributes, "data-name")) &&
      std::string(attribute->value) == "LinkArea") {
      result.push_back(node);
  }

  GumboVector* children = &node->v.element.children;
  for (unsigned int i = 0; i < children->length; ++i) {
      findLinkAreaTags(static_cast<GumboNode*>(children->data[i]), result);
  }
}

const char* GetAttribute(const GumboElement& element, const char* attr_name) {
  auto attributes = &element.attributes;
  for (unsigned int i = 0; i < attributes->length; ++i) {
      GumboAttribute* attribute = static_cast<GumboAttribute*>(attributes->data[i]);
      if (strcmp(attribute->name, attr_name) == 0) {
          return attribute->value;
      }
  }
  return nullptr;
}

std::string FindHrefInNode(const GumboNode* node) {
  if (node->type != GUMBO_NODE_ELEMENT || node->v.element.tag != GUMBO_TAG_A) {
      return "";
  }

  const GumboElement& element = node->v.element;
  const char* class_attr = GetAttribute(element, "class");

  if (class_attr && strcmp(class_attr, "_93444fe79c--link--eoxce") == 0) {
      const char* href = GetAttribute(element, "href");
      return href ? href : "";
  }

  return "";
}



std::string TraverseAndFindHref(const GumboNode* node) {
  if (node->type != GUMBO_NODE_ELEMENT && node->type != GUMBO_NODE_DOCUMENT) {
      return "";
  }

  std::string href = FindHrefInNode(node);
  if (!href.empty()) {
      return href;
  }

  const GumboVector* children = &node->v.element.children;
  for (unsigned int i = 0; i < children->length; ++i) {
      const GumboNode* child = static_cast<const GumboNode*>(children->data[i]);
      href = TraverseAndFindHref(child);
      if (!href.empty()) {
          return href;
      }
  }

  return "";
}


std::vector<std::string> ParseHrefFromResponse(const std::string& response) {
  GumboOutput* output = gumbo_parse(response.c_str());
  std::vector<GumboNode*> nodes;
  findLinkAreaTags(output->root, nodes);
  std::vector<std::string> result;
  for (size_t idx = 0; idx < nodes.size(); ++idx) {
    if (idx % 2 == 1) {
      continue;
    }
    result.emplace_back(TraverseAndFindHref(nodes[idx]));
  }
  return result;
}

namespace realty-parser {

namespace {
  class LatestFlatsListener final : public userver::components::ComponentBase {
    public:
        static constexpr std::string_view kName = "latest-flats-listener-component";
    
        LatestFlatsListener(const userver::components::ComponentConfig& config,
                         const userver::components::ComponentContext& context)
            : userver::components::ComponentBase(config, context)
            , client_(context.FindComponent<userver::components::HttpClient>().GetHttpClient())
            , s3_client_(context.FindComponent<realty-parser::components::S3ApiComponent>().GetClient())
            , pg_cluster_(context.FindComponent<userver::components::Postgres>("postgres-db").GetCluster())
            , telegram_bot_(context.FindComponent<realty-parser::components::TelegramBotComponent>().GetTelegramBot())
            , config_(config) {

            request_timeout_ = config["request_timeout"].As<int>();
            sleep_before_request_ = config["sleep_before_request"].As<int>();
            sleep_timeout_ = config["sleep_timeout"].As<int>();
            max_tries_ = config["max_tries"].As<int>();
            sleep_multiplier_ = config["sleep_multiplier"].As<double>();
            debug_logs_ = config["debug_logs"].As<bool>();
            cookie_header_ = config["cookie_header"].As<std::string>();
            task_mode_ = config["task_mode"].As<std::string>();

            auto period = config["task_period_seconds"].As<int>();
            periodic_task_.Start("latest-flats-listener", std::chrono::seconds(period), [this] {
                ExecuteTask();
            });
        }
    
        ~LatestFlatsListener() {
            periodic_task_.Stop();
        }
    
        void ExecuteTask() {
            FlatSearchFilter flatsFilter{
              config_["search_params"]["min_price"].As<int>(),
              config_["search_params"]["max_price"].As<int>(),
              config_["search_params"]["min_area"].As<int>(),
              config_["search_params"]["max_area"].As<int>(),
              config_["search_params"]["sort"].As<std::string>(),
              config_["search_params"]["room1"].As<int>(),
              config_["search_params"]["room2"].As<int>(),
              config_["search_params"]["room3"].As<int>(),
              config_["search_params"]["room4"].As<int>(),
              config_["search_params"]["room5"].As<int>(),
              config_["search_params"]["room6"].As<int>(),
              config_["search_params"]["pages_limit"].As<int>(),
              config_["search_params"]["limit"].As<int>()
            };
            int pageslimit = config_["search_params"]["pages_limit"].As<int>();
            int overallLimit = config_["search_params"]["overall_limit"].As<int>();
            int parsed = 0;
            std::string flatsList = FlatInfo::HeadersList();
            std::vector<FlatInfo> infos;
            for (int page = 1; parsed < overallLimit && page <= pageslimit; ++page) {
              try {
                if (task_mode_ == "actual_flats") {
                  flatsFilter.page = 1;
                } else if (task_mode_ == "s3_upload") {
                  flatsFilter.page = page;
                }
                auto request = client_.CreateRequest()
                .get(GetFlatSearchUrl(flatsFilter))
                .http_version(userver::http::HttpVersion::k11)
                .timeout(request_timeout_);
                std::cerr << "sending main request\n";
                auto response = request.perform();
                if (response->IsOk()) {
                    std::vector<std::string> hrefs = ParseHrefFromResponse(response->body());
                    if (debug_logs_) {
                      std::cerr << "href size = " << hrefs.size() << '\n';
                    }
                    for (auto& href : hrefs) {
                      if (parsed >= overallLimit) {
                        continue;
                      }
                      if (task_mode_ == "actual_flats") {
                        auto result = pg_cluster_->Execute(
                          userver::storages::postgres::ClusterHostType::kMaster,
                          "SELECT href FROM user_sent_messages where href = '" + href + "';");
                        if (result.Size() != 0) {
                          LOG_INFO() <<  "this flat already sent | href = " << href;
                          continue;
                        }
                      }
                      auto flatInfo = GetFlatInfo(href, client_, {request_timeout_, sleep_before_request_, sleep_timeout_, sleep_multiplier_, max_tries_, debug_logs_, cookie_header_});
                      ++parsed;
                      if (task_mode_ == "actual_flats") {
                        infos.push_back(flatInfo);
                      } else if (task_mode_ == "s3_upload") {
                        flatsList += flatInfo.ToCSVString();
                      }
                    }
                }
              } catch (const std::exception& e) {
                  LOG_ERROR() << "MYLOG Exception during HTTP request: " << e.what();
              }
            }
            if (task_mode_ == "actual_flats") {
              auto sentCount = telegram_utils::SendFlatInfoMessages(telegram_bot_, pg_cluster_, infos);
              std::cerr << "sent " << sentCount << " messages\n";
            } else if (task_mode_ == "s3_upload") {
              if (debug_logs_) {
                std::cerr << "moved " << parsed << " flats to storage\n";
              }
              s3_client_->PutObject("flats_list.csv", flatsList);
            }
        }

        static userver::yaml_config::Schema GetStaticConfigSchema() {
          return userver::yaml_config::MergeSchemas<ComponentBase>(R"(
          type: object
          description: schema for latest_flats_listener
          additionalProperties: false
          properties:
            task_period_seconds:
              description: task_period_seconds
              type: integer
            request_timeout:
              description: request timeout in milliseconds
              type: integer
            sleep_before_request:
              description: sleep in milliseconds before first request
              type: integer
            sleep_timeout:
              description: sleep_timeout in milliseconds
              type: integer
            sleep_multiplier:
              description: sleep_multiplier for request
              type: number
            max_tries:
              description: max_tries for request
              type: integer
            debug_logs:
              description: write debug logs using cerr
              type: boolean
            cookie_header:
              description: cookie header for requests
              type: string
            task_mode:
              description: task_mode for results
              type: string
            search_params:
              additionalProperties: false
              description: search parameters for the flat requests
              type: object
              properties:
                min_price:
                  description: minimal price of the flat
                  type: integer
                max_price:
                  description: maximal price of the flat
                  type: integer
                min_area:
                  description: minimal area of the flat
                  type: integer
                max_area:
                  description: maximal area of the flat
                  type: integer
                sort:
                  description: sorting order of flats
                  type: string
                room1:
                  description: include 1-room flats
                  type: integer
                room2:
                  description: include 2-room flats
                  type: integer
                room3:
                  description: include 3-room flats
                  type: integer
                room4:
                  description: include 4-room flats
                  type: integer
                room5:
                  description: include 5-room flats
                  type: integer
                room6:
                  description: include 6+ room flats
                  type: integer
                pages_limit:
                  description: pages limit for search
                  type: integer
                limit:
                  description: limit of flats per page
                  type: integer
                overall_limit:
                  description: overall limit for search
                  type: integer
          )");
        }
    
    private:
        userver::utils::PeriodicTask periodic_task_;
        userver::clients::http::Client& client_;
        userver::s3api::ClientPtr s3_client_;
        userver::storages::postgres::ClusterPtr pg_cluster_;
        TgBot::Bot& telegram_bot_;
        const userver::components::ComponentConfig& config_;

        int request_timeout_;
        int sleep_before_request_;
        int sleep_timeout_;
        double sleep_multiplier_;
        int max_tries_;
        bool debug_logs_;
        std::string cookie_header_;
        std::string task_mode_;
    };
}  // namespace


void EnableLatestFlatsListener(userver::components::ComponentList& component_list) {
  component_list.Append<LatestFlatsListener>();
}

}  // namespace realty-parser
