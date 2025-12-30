#pragma once

#include "flat_info.hpp"

#include <tgbot/tgbot.h>
#include <userver/storages/postgres/cluster.hpp>
#include <userver/storages/postgres/component.hpp>

namespace realty-parser {
    namespace telegram_utils {
        int SendFlatInfoMessages(TgBot::Bot& bot, userver::storages::postgres::ClusterPtr cluster, const std::vector<FlatInfo>& infos) {
            auto result = cluster->Execute(
                userver::storages::postgres::ClusterHostType::kMaster,
                "SELECT * FROM user_subscriptions");
            int count = 0;

            for (auto& row : result) {
                std::int64_t chatId = row["chat_id"].As<std::int64_t>();
                if (chatId == 0) {
                    std::cerr << "chat id is empty\n";
                    continue;
                }
                for (auto& flat : infos) {
                    if (
                        flat.price > row["max_price"].As<int>() ||
                        flat.price < row["min_price"].As<int>() ||
                        flat.total_area < row["min_area"].As<int>() ||
                        flat.room_number < row["min_room"].As<int>() ||
                        flat.room_number > row["max_room"].As<int>() ||
                        flat.flat_score < row["minimal_score"].As<int>() ||
                        flat.underground_place < row["minimal_underground_place"].As<int>() ||
                        flat.underground_place > row["maximal_underground_place"].As<int>() ||
                        time_utils::GetCurrentTimestamp() < row["subscription_from"].As<std::int64_t>() ||
                        time_utils::GetCurrentTimestamp() > row["subscription_to"].As<std::int64_t>()
                    ) {
                        std::cerr << "skipping flat because of filters | " << flat.ToCSVString() << " | chat id = " << chatId << '\n';
                        continue;
                    }
                    std::string selectQuery = "select * from user_sent_messages where chat_id = " + std::to_string(chatId) + " and href = '" + flat.link + "'";
                    auto selectResult = cluster->Execute(userver::storages::postgres::ClusterHostType::kMaster, selectQuery);
                    if (selectResult.Size() != 0) {
                        std::cerr << "skipping because flat already sent | link = " << flat.link << " | chat id = " << chatId << '\n';
                        continue;
                    }
                    // TO DO: add batch dedublication
                    ++count;
                    bot.getApi().sendMessage(chatId, flat.ToTelegramMessage());
                    std::string queryStr = flat.UserSentMessagesInsertQuery(std::to_string(chatId));
                    auto queryResult = cluster->Execute(userver::storages::postgres::ClusterHostType::kMaster, queryStr);
                }
            }
            return count;
        }
    }
}

