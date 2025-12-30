#include <userver/clients/http/component.hpp>
#include <userver/components/loggable_component_base.hpp>

#include <userver/yaml_config/merge_schemas.hpp>
#include <userver/s3api/clients/fwd.hpp>

namespace realty-parser::components {

class S3ApiComponent : public userver::components::ComponentBase {
public:
    static constexpr std::string_view kName = "s3-component";

    S3ApiComponent(const userver::components::ComponentConfig& config, const userver::components::ComponentContext& context);

    userver::s3api::ClientPtr RegisterClient(const userver::components::ComponentConfig& config);

    userver::s3api::ClientPtr GetClient();

    static userver::yaml_config::Schema GetStaticConfigSchema() {
        return userver::yaml_config::MergeSchemas<ComponentBase>(R"(
        type: object
        description: schema for s3_api_component
        additionalProperties: false
        properties:
          storage_domain:
            description: domain for storage operations
            type: string
          storage_access_key:
            description: access key for S3 storage
            type: string
          storage_secret_key:
            description: secret key for S3 storage
            type: string
          bucket_name:
            description: bucket name for S3 storage
            type: string
        )");
    }

private:
    userver::clients::http::Client& http_client_;
    userver::s3api::ClientPtr api_client_;
};

}  // namespace realty-parser::components
