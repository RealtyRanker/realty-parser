#pragma once

#include <userver/clients/http/client.hpp>
#include <userver/clients/http/component.hpp>
#include <userver/clients/http/request.hpp>
#include <userver/engine/sleep.hpp>
#include <userver/formats/json.hpp>


#include "flat_info.hpp"
#include "flat_score.hpp"
#include "time_utils.hpp"

namespace realty-parser {
    struct FlatRequestSettings {
        int request_timeout;
        int sleep_before_request;
        int sleep_timeout;
        double sleep_multiplier;
        int max_tries;
        bool debug_logs;
        std::string cookie_header;
    };

    enum class EValueType {
        String = 1,
        Int = 2,
        Double = 3,
        Boolean = 4,
        Json = 5
    };

    using JsonValueVariant = std::variant<std::string, int, double, bool, userver::formats::json::Value>;
    std::string StringUNDEF("UNDEF");
    constexpr int IntUNDEF = 0;
    constexpr double DoubleUNDEF = 0.0;
    constexpr bool BoolUNDEF = false;

    JsonValueVariant UndefValue(const EValueType type) {
        switch (type) {
            case realty-parser::EValueType::String:
                return StringUNDEF;
            case realty-parser::EValueType::Int:
                return IntUNDEF;
            case realty-parser::EValueType::Double:
                return DoubleUNDEF;
            case realty-parser::EValueType::Boolean:
                return BoolUNDEF;
            case realty-parser::EValueType::Json:
                return userver::formats::json::Value();
        }
        return StringUNDEF;
    }

    double StrToDouble(const std::string& str) {
        try {
            return std::stod(str);
        } catch (...) {
            return 0.0;
        }
    }

    JsonValueVariant GetJsonValue(const userver::formats::json::Value& json, const std::vector<std::string>& path, const EValueType type) {
        auto cur = json;
        for (auto& p : path) {
            if (cur[p].IsNull()) {
                return UndefValue(type);
            }
            cur = cur[p];
        }
        switch (type) {
            case realty-parser::EValueType::String:
                return (cur.IsString() ? cur.As<std::string>() : UndefValue(type));
            case realty-parser::EValueType::Int:
                return (cur.IsInt64() ? cur.As<int>() : UndefValue(type));
            case realty-parser::EValueType::Double:
                return (cur.IsDouble() ? cur.As<double>() : (cur.IsString() ? StrToDouble(cur.As<std::string>()) : UndefValue(type)));
            case realty-parser::EValueType::Boolean:
                return (cur.IsBool() ? cur.As<bool>() : UndefValue(type));
            case realty-parser::EValueType::Json:
                return cur;
        }
        return UndefValue(type);
    }

    std::string ExtractStringValue(const userver::formats::json::Value& json, const std::vector<std::string>& path) {
        JsonValueVariant value = GetJsonValue(json, path, realty-parser::EValueType::String);
        return (std::holds_alternative<std::string>(value) ? std::get<std::string>(value) : StringUNDEF);
    }

    int ExtractIntValue(const userver::formats::json::Value& json, const std::vector<std::string>& path) {
        JsonValueVariant value = GetJsonValue(json, path, realty-parser::EValueType::Int);
        return (std::holds_alternative<int>(value) ? std::get<int>(value) : IntUNDEF);
    }

    double ExtractDoubleValue(const userver::formats::json::Value& json, const std::vector<std::string>& path) {
        JsonValueVariant value = GetJsonValue(json, path, realty-parser::EValueType::Double);
        return (std::holds_alternative<double>(value) ? std::get<double>(value) : DoubleUNDEF);
    }

    bool ExtractBoolValue(const userver::formats::json::Value& json, const std::vector<std::string>& path) {
        JsonValueVariant value = GetJsonValue(json, path, realty-parser::EValueType::Boolean);
        return (std::holds_alternative<bool>(value) ? std::get<bool>(value) : BoolUNDEF);
    }

    userver::formats::json::Value ExtractJsonValue(const userver::formats::json::Value& json, const std::vector<std::string>& path) {
        JsonValueVariant value = GetJsonValue(json, path, realty-parser::EValueType::Json);
        return (std::holds_alternative<userver::formats::json::Value>(value) ? std::get<userver::formats::json::Value>(value) : userver::formats::json::Value());
    }

    FlatInfo ParseRawFlatInfo(const userver::formats::json::Value& json, const std::string& href, bool debugLogs) {
        FlatInfo info;
        info.link = href;
        info.price = ExtractIntValue(json, {"value", "offerData", "offer", "bargainTerms", "price"});
        info.room_number = ExtractIntValue(json, {"value", "offerData", "offer", "roomsCount"});
        info.total_area = ExtractDoubleValue(json, {"value", "offerData", "offer", "totalArea"});
        info.living_area = ExtractDoubleValue(json, {"value", "offerData", "offer", "livingArea"});
        info.kitchen_area = ExtractDoubleValue(json, {"value", "offerData", "offer", "kitchenArea"});
        info.floor = ExtractIntValue(json, {"value", "offerData", "offer", "floorNumber"});
        info.max_floor = ExtractIntValue(json, {"value", "offerData", "offer", "building", "floorsCount"});
        info.comission = ExtractIntValue(json, {"value", "offerData", "offer", "bargainTerms", "agentFee"});
        info.deposit = ExtractIntValue(json, {"value", "offerData", "offer", "bargainTerms", "deposit"});
        info.deposit_months = ExtractIntValue(json, {"value", "offerData", "offer", "bargainTerms", "prepayMonths"});
        info.renovation = ExtractStringValue(json, {"value", "offerData", "offer", "repairType"});
        info.is_apartments = ExtractBoolValue(json, {"value", "offerData", "offer", "isApartments"});
        info.loggia_count = ExtractIntValue(json, {"value", "offerData", "offer", "loggiasCount"});
        info.balcony_count = ExtractIntValue(json, {"value", "offerData", "offer", "balconiesCount"});
        info.windows_view = ExtractStringValue(json, {"value", "offerData", "offer", "windowsViewType"});
        info.separated_bathroom_count = ExtractIntValue(json, {"value", "offerData", "offer", "separateWcsCount"});
        info.combined_bathroom_count = ExtractIntValue(json, {"value", "offerData", "offer", "combinedWcsCount"});
        info.has_dishwasher = ExtractBoolValue(json, {"value", "offerData", "offer", "hasDishwasher"});
        info.has_conditioner = ExtractBoolValue(json, {"value", "offerData", "offer", "hasConditioner"});
        info.children_allowed = ExtractBoolValue(json, {"value", "offerData", "offer", "childrenAllowed"});
        info.pets_allowed = ExtractBoolValue(json, {"value", "offerData", "offer", "petsAllowed"});
        info.last_updated = ExtractStringValue(json, {"value", "offerData", "offer", "humanizedEditDate"});
        info.ceiling_height = ExtractDoubleValue(json, {"value", "offerData", "offer", "building", "ceilingHeight"});
        info.building_entrances_number = ExtractIntValue(json, {"value", "offerData", "bti", "houseData", "entrances"});
        info.building_apartments_number = ExtractIntValue(json, {"value", "offerData", "bti", "houseData", "flatCount"});
        info.building_elevators_number = ExtractIntValue(json, {"value", "offerData", "bti", "houseData", "lifts"});

        info.FillUndergroundInfo(ExtractJsonValue(json, {"value", "offerData", "offer", "geo", "undergrounds"}));

        info.last_updated_timestamp = time_utils::GetCurrentTimestamp();
        info.last_updated_string = time_utils::GetCurrentTimeStr();
        info.flat_score = flat_scoring_utils::CalculateFlatScore(info, debugLogs);
        return info;
    }

    FlatInfo ParseOfferCard(const std::string& str, const std::string& href, bool debugLogs) {
        const std::string prefix_str = "window._cianConfig['frontend-offer-card'] = (window._cianConfig['frontend-offer-card'] || []).concat(";
        if (debugLogs) {
            std::cerr << "str size = " << str.size() << " | time = " << time_utils::GetCurrentTimeStr() << '\n';
        }
        if (str.size() < prefix_str.size() + 2) {
            return FlatInfo();
        }
        auto strCopy = str.substr(0, str.size() - 2);
        strCopy = strCopy.substr(prefix_str.size());
        userver::formats::json::Value json = userver::formats::json::FromString(strCopy);
        if (!json.IsArray()) {
            std::cerr << "Not an array";
            return FlatInfo();
        }
        userver::formats::json::Value flatInfoJson;

        for (auto& value : json) {
            if (value["key"].IsString() && value["key"].As<std::string>() == "defaultState") {
                return ParseRawFlatInfo(value, href, debugLogs);
            }
        }
        return FlatInfo();
    }

    FlatInfo ParseInfoFromResponse(const std::string& response, const std::string& href, bool debugLogs) {
        std::istringstream stream(response);
        std::string line;
        while (std::getline(stream, line)) {
            if (line.find("window._cianConfig['frontend-offer-card']") != std::string::npos) {
                return ParseOfferCard(line, href, debugLogs);
            }
        }
        return FlatInfo();
    }

    FlatInfo GetFlatInfo(const std::string& href, userver::clients::http::Client& client, const FlatRequestSettings& settings) {
        userver::engine::SleepFor(std::chrono::milliseconds{settings.sleep_before_request});
        auto request = client.CreateRequest()
        .get(href)
        .http_version(userver::http::HttpVersion::k11)
        .timeout(settings.request_timeout)
        .headers({
            {"Cookie", settings.cookie_header}
        });
        auto response = request.perform();
        int tries = 0;
        int cur_sleep_timeout = settings.sleep_timeout;
        while (tries < settings.max_tries && (!response->IsOk() || response->body().find("rate_limit") != std::string::npos)) {
            ++tries;
            if (!response->IsOk()) {
                std::cerr << "response is not OK, response code = " << response->status_code() << '\n';
            } else {
                std::cerr << "rate_limit in response body\n";
            }
            userver::engine::SleepFor(std::chrono::milliseconds{cur_sleep_timeout});
            cur_sleep_timeout = int((double)cur_sleep_timeout * settings.sleep_multiplier);
            response = request.perform();
        }
        if (!response->IsOk()) {
            return FlatInfo();
        }
        return ParseInfoFromResponse(response->body(), href, settings.debug_logs);
    }
}