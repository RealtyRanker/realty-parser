#include <userver/components/loggable_component_base.hpp>
#include <tgbot/tgbot.h>


#include <userver/yaml_config/merge_schemas.hpp>

namespace realty-parser::components {

class TelegramBotComponent : public userver::components::ComponentBase {
public:
    static constexpr std::string_view kName = "telegram-bot-component";

    TelegramBotComponent(const userver::components::ComponentConfig& config, const userver::components::ComponentContext& context);

    TgBot::Bot& GetTelegramBot();

    static userver::yaml_config::Schema GetStaticConfigSchema() {
        return userver::yaml_config::MergeSchemas<ComponentBase>(R"(
        type: object
        description: schema for telegram_bot component
        additionalProperties: false
        properties:
          bot_token:
            description: token for telegram bot
            type: string
        )");
    }

private:
    TgBot::Bot bot_;
};

}  // namespace realty-parser::components
