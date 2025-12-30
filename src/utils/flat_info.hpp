#pragma once

#include "subway_scoring_utils.hpp"
#include "time_utils.hpp"

#include <bits/stdc++.h>

#include <boost/pfr.hpp>

#include <userver/formats/json.hpp>

namespace realty-parser {

    std::string AttributeToSql(int attr) {
        return ", " + std::to_string(attr);
    }

    std::string AttributeToSql(double attr) {
        return ", " + std::to_string(attr);
    }

    std::string AttributeToSql(const std::string& attr) {
        return ", '" + attr + "'";
    }

    std::string AttributeToSql(bool attr) {
        return (attr ? ", TRUE" : ", FALSE");
    }
    

    struct FlatInfo {
        std::string link;
        int price;
        int flat_score;
        double underground_score;
        int underground_place;
        std::string underground_distance_info;
        int room_number;
        double total_area;
        double living_area;
        double kitchen_area;
        int floor;
        int max_floor;
        int deposit;
        int deposit_months;
        int comission;
        std::string renovation;
        bool is_apartments;
        int loggia_count;
        int balcony_count;
        std::string windows_view;
        int separated_bathroom_count;
        int combined_bathroom_count;
        bool has_dishwasher;
        bool has_conditioner;
        bool children_allowed;
        bool pets_allowed;
        std::string last_updated;
        double ceiling_height;
        int building_entrances_number;
        int building_apartments_number;
        int building_elevators_number;
        int last_updated_timestamp;
        std::string last_updated_string;

        std::string ToCSVString() const {
            std::string result;
            result += link + ";" + std::to_string(price) + ";" + std::to_string(flat_score) + ";";
            result += std::to_string(underground_score) + ";" + std::to_string(underground_place) + ";" + underground_distance_info + ";";
            result += std::to_string(room_number) + ";";
            result += std::to_string(total_area) + ";" + std::to_string(living_area) + ";" + std::to_string(kitchen_area) + ";";
            result += std::to_string(floor) + ";" + std::to_string(max_floor) + ";" + std::to_string(deposit) + ";";
            result += std::to_string(comission) + ";" + std::to_string(deposit_months) + ";" + renovation + ";" + std::to_string(is_apartments) + ";";
            result += std::to_string(loggia_count) + ";" + std::to_string(balcony_count) + ";" + windows_view + ";";
            result += std::to_string(separated_bathroom_count) + ";" + std::to_string(combined_bathroom_count) + ";" + std::to_string(has_dishwasher) + ";";
            result += std::to_string(has_conditioner) + ";" + std::to_string(children_allowed) + ";" + std::to_string(pets_allowed) + ";";
            result += last_updated + ";" + std::to_string(ceiling_height) + ";" + std::to_string(building_entrances_number) + ";";
            result += std::to_string(building_apartments_number) + ";" + std::to_string(building_elevators_number) + ";";
            result += std::to_string(last_updated_timestamp) + ";" + last_updated_string + '\n';
            return result;
        }

        std::string ToTelegramMessage() const {
            std::string result;
            result += "Ссылка: " + link + '\n';
            result += "Цена: " + std::to_string(price) + '\n';
            result += "Станции: " + underground_distance_info + '\n';
            result += "Место станции метро: " + std::to_string(underground_place) + '\n';
            result += "Количество комнат: " + std::to_string(room_number) + '\n';
            result += "Общая площадь: " + std::to_string(int(total_area)) + '\n';
            result += "Жилая площадь: " + std::to_string(int(living_area)) + '\n';
            result += "Площадь кухни: " + std::to_string(int(kitchen_area)) + '\n';
            result += "Ремонт: " + renovation + '\n';
            result += "Этаж: " + std::to_string(floor) + " из " + std::to_string(max_floor) + '\n';
            result += std::to_string(loggia_count) + " лоджий, " + std::to_string(balcony_count) + " балконов\n";
            result += std::to_string(separated_bathroom_count) + " совмещенных с/у, " + std::to_string(combined_bathroom_count) + " раздельных с/у\n";
            result += (has_dishwasher ? "Посудомойка: есть\n" : "Посудомойка: нет\n");
            result += (has_conditioner ? "Кондиционер: есть\n" : "Кондиционер: нет\n");
            result += (pets_allowed ? "Можно с животными: ДА\n" : "Можно с животными: НЕТ\n");
            result += "Score: " + std::to_string(flat_score) + '\n';
            return result;
        }

        std::string UserSentMessagesInsertQuery(const std::string& chatId) const {
            std::string queryStr = "INSERT INTO user_sent_messages VALUES(" + chatId + ", " + std::to_string(time_utils::GetCurrentTimestamp());
            queryStr += AttributeToSql(link) + AttributeToSql(price) + AttributeToSql(flat_score);
            queryStr += AttributeToSql(underground_score) + AttributeToSql(underground_place) + AttributeToSql(underground_distance_info);
            queryStr += AttributeToSql(room_number) + AttributeToSql(total_area) + AttributeToSql(living_area);
            queryStr += AttributeToSql(kitchen_area) + AttributeToSql(floor) + AttributeToSql(max_floor);
            queryStr += AttributeToSql(deposit) + AttributeToSql(deposit_months) + AttributeToSql(comission);
            queryStr += AttributeToSql(renovation) + AttributeToSql(is_apartments) + AttributeToSql(loggia_count);
            queryStr += AttributeToSql(balcony_count) + AttributeToSql(windows_view) + AttributeToSql(separated_bathroom_count);
            queryStr += AttributeToSql(combined_bathroom_count) + AttributeToSql(has_dishwasher) + AttributeToSql(has_conditioner);
            queryStr += AttributeToSql(children_allowed) + AttributeToSql(pets_allowed) + AttributeToSql(last_updated);
            queryStr += AttributeToSql(ceiling_height) + AttributeToSql(building_entrances_number) + AttributeToSql(building_apartments_number);
            queryStr += AttributeToSql(building_elevators_number) + AttributeToSql(last_updated_timestamp) + AttributeToSql(last_updated_string);
            
            queryStr += ")";
            std::cerr << queryStr << '\n';
            return queryStr;
        }

        static std::string HeadersList() {
            return std::string("Ссылка;Цена;Flat Score;Underground Score;Underground Place;Близость к метро;Количество комнат;Общая площадь;Жилая площадь") + 
            std::string(";Площадь кухни;Этаж;Количество этажей;Депозит;Комиссия;Месяцы предоплаты;Ремонт;Апартаменты;Лоджии;Балконы;Окна;Разделенные с/у;Совмещенные с/у;Посудомойка;Кондиционер") +
            std::string(";Можно с детьми;Можно с животными;Обновлено;Высота потолков;Количество подъездов;Количество квартир в доме;Количество лифтов в доме;Timestamp обработки;Дата обработки\n");
        }

        void FillUndergroundInfo(const userver::formats::json::Value& json) {
            // underground_score - счет станции
            // underground_place - номер станции в топе
            // underground_distance_info - текстовая информация о расстоянии до станций
            int best_station_place = subway_scoring::undefined_place;
            double best_station_score = subway_scoring::undefined_score;
            std::string distance_info;
            int stations_count = 0;
            for (auto& station : json) {
                ++stations_count;
                auto info = subway_scoring::GetSubwayInfo(station["name"].As<std::string>());
                auto score = info.subway_score;
                if (station["travelType"].As<std::string>() == "walk") {
                    score += 1.1 * station["travelTime"].As<double>();
                } else {
                    score += 7 * station["travelTime"].As<int>();
                }
                if (score < best_station_score) {
                    best_station_score = score;
                    best_station_place = info.subway_place;
                }
                distance_info += subway_scoring::GetSubwayInfoString(station);
            }
            underground_score = best_station_score - (stations_count - 1) * subway_scoring::additional_station_multiplier;
            underground_place = best_station_place;
            underground_distance_info = distance_info;
        }
    };
}