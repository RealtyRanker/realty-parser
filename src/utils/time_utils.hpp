#pragma once

#include <bits/stdc++.h>

#include <userver/utils/datetime.hpp>

namespace realty-parser {
    namespace time_utils {
        std::string GetCurrentTimeStr() {
            auto now = userver::utils::datetime::Now();
            std::time_t timeT = std::chrono::system_clock::to_time_t(now);
            std::tm tm = *std::localtime(&timeT);
            std::ostringstream oss;
            oss << std::put_time(&tm, "%Y-%m-%d %H:%M:%S");
            return oss.str();
        }

        int GetCurrentTimestamp() {
            auto now = userver::utils::datetime::Now();
            auto epoch_duration = now.time_since_epoch();
            return std::chrono::duration_cast<std::chrono::seconds>(epoch_duration).count();
        }
    }
}