#include "telegram_bot.hpp"

#include <userver/components/component_config.hpp>
#include <userver/components/component_context.hpp>

namespace realty-parser::components {

    TelegramBotComponent::TelegramBotComponent(
        const userver::components::ComponentConfig& config,
        const userver::components::ComponentContext& context
    ) 
    : ComponentBase(config, context)
    , bot_(config["bot_token"].As<std::string>())
    {
/*
        bot_.getEvents().onCommand("start", [this](TgBot::Message::Ptr message) {
            bot_.getApi().sendMessage(message->chat->id, "Your chat ID = " + std::to_string(message->chat->id));
        });

        try {
            printf("Bot username: %s\n", bot_.getApi().getMe()->username.c_str());
            TgBot::TgLongPoll longPoll(bot_);
            longPoll.start();
        } catch (TgBot::TgException& e) {
            printf("error: %s\n", e.what());
        }
        */
    }

    TgBot::Bot& TelegramBotComponent::GetTelegramBot() {
        return bot_;
    }

}  // namespace realty-parser::components
