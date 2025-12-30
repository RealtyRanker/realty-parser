#include "s3api_client.hpp"

#include <userver/components/component_config.hpp>
#include <userver/components/component_context.hpp>

#include <userver/s3api/authenticators/access_key.hpp>
#include <userver/s3api/clients/s3api.hpp>
#include <userver/s3api/models/s3api_connection_type.hpp>


namespace realty-parser::components {

    S3ApiComponent::S3ApiComponent(
    const userver::components::ComponentConfig& config,
    const userver::components::ComponentContext& context
)
    : ComponentBase(config, context),
      http_client_(context.FindComponent<userver::components::HttpClient>().GetHttpClient()) {
    api_client_ = RegisterClient(config);
}

userver::s3api::ClientPtr S3ApiComponent::RegisterClient(const userver::components::ComponentConfig& config) {
    auto connection_cfg = userver::s3api::ConnectionCfg(
        std::chrono::milliseconds{1000}, /* timeout */
        2,                              /* retries */
        std::nullopt                    /* proxy */
    );

    auto s3_connection = userver::s3api::MakeS3Connection(
        http_client_, userver::s3api::S3ConnectionType::kHttps, config["storage_domain"].As<std::string>(), connection_cfg
    );

    auto auth = std::make_shared<userver::s3api::authenticators::AccessKey>(config["storage_access_key"].As<std::string>(), userver::s3api::Secret(config["storage_secret_key"].As<std::string>()));

    return userver::s3api::GetS3Client(s3_connection, auth, config["bucket_name"].As<std::string>());
}

userver::s3api::ClientPtr S3ApiComponent::GetClient() {
    return api_client_;
}

}  // namespace realty-parser::components
