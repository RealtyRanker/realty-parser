#pragma once

#include <userver/components/component_base.hpp>
#include <userver/components/component_context.hpp>
#include <userver/components/component.hpp>
#include <userver/components/statistics_storage.hpp>
#include <userver/formats/json/value.hpp>
#include <userver/logging/log.hpp>
#include <userver/utils/async.hpp>
#include <userver/utils/statistics/storage.hpp>
#include <userver/utils/statistics/writer.hpp>

namespace realty-parser::components {

class Statistics final : public userver::components::LoggableComponentBase {
public:
    constexpr static const char* kName = "statistics-component";

    Statistics(const userver::components::ComponentConfig& config,
                           const userver::components::ComponentContext& context)
        : userver::components::LoggableComponentBase(config, context)
    {
        auto& statistics_storage = context.FindComponent<userver::components::StatisticsStorage>().GetStorage();
        extender_holder_ = statistics_storage.RegisterExtender(
            kName, [this](const userver::utils::statistics::StatisticsRequest&) {
                return metrics_;
            });
    }

    ~Statistics() override {
        extender_holder_.Unregister();
    }

    void AddMetric(const std::string& key, double value) const {
      LOG_INFO() << "MYLOG key = " << key << " | value = " << std::to_string(value);
      metrics_[key] = value;
    }

private:
    userver::utils::statistics::Entry extender_holder_;
    mutable userver::formats::json::ValueBuilder metrics_{userver::formats::common::Type::kObject};
};

}  // namespace realty-parser::components
