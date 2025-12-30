#pragma once

#include "flat_info.hpp"

namespace realty-parser {
    namespace flat_scoring_utils {
        const double default_score_value = 600000.0;
        const double eps = 1e-5;

        const double deposit_multiplier = 0.02; // Коэффициент за каждый рубль депозита
        const double all_area_multiplier = 300.0; // Коэффициент за каждый м^2 общей площади
        const double cost_multiplier = 1.0; // Коэффициент за каждый рубль цены
        const int max_possible_price = 300000; // Максимально возможная цена
        const double living_area_multiplier = 600.0; // Коэффициент за каждый м^2 жилой площади
        const double living_area_part = 0.5; // Коэффициент средней жилой площади от общей
        const double kitchen_area_multiplier = 50.0; // Коэффициент за каждый м^2 площади кухни
        const double max_kitchen_area = 50.0; // Максимально возможная площадь кухни
        const double default_kitchen_area = 5.0; // Значение площади кухни по умолчанию
        const double comission_multiplier = 0.0006; // Коэффициент за каждый процент комиссии
        const double prepaid_months_multiplier = 3000.0; // Коэффициент за каждый месяц предоплаты
        const double children_multiplier = 0.0; // Коэффициент за возможность с детьми
        const double pets_multiplier = 4000.0; // Коэффициент за возможность с животными
        const double dishwasher_multiplier = 1500.0; // Коэффициент за наличие посудомойки
        const double conditioner_multiplier = 1500.0; // Коэффициент за наличие кондиционера
        const double apartments_multiplier = 3000.0; // Коэффициент за апартаменты
        const double two_room_multiplier = 3000.0; // Коэффициент за 2-комнатность
        const double three_room_multiplier = 5000.0; // Коэффициент за 3-комнатность
        const double four_room_multiplier = 7500.0; // Коэффициент за 4-комнатность
        const double additional_rooms_multiplier = 1500.0; // Коэффициент за каждую комнату выше 4
        const double windows_yard_multiplier = 400.0; // Коэффициент за окна во двор
        const double windows_street_multiplier = 200.0; // Коэффициент за окна на улицу
        const double windows_both_multiplier = 500.0; // Коэффициент за окна на улицу и во двор
        const double renovation_cosmetic_multiplier = 3000.0; // Коэффициент за наличие косметического ремонта
        const double renovation_euro_multiplier = 5500.0; // Коэффициент за наличие евро-ремонта
        const double renovation_design_multiplier = 7000.0; // Коэффициент за наличие дизайнерского ремонта
        const double bathroom_combined_multiplier = 0.0; // Коэффициент за каждый совмещенный с/у
        const double bathroom_separated_multiplier = 1000.0; // Коэффициент закаждый раздельный с/у
        const double balconies_balcony_multiplier = 1500.0; // Коэффициент за каждый балкон
        const double balconies_loggia_multiplier = 2500.0; // Коэффициент за каждую лоджию
        const double default_underground_score = 50.0; // Значение счета метро по умолчанию
        const double underground_score_multiplier = 1000.0; // Коэффициент за каждую единицу счета метро


        double CalculateDepositScore(FlatInfo info) {
            return -deposit_multiplier * double(info.deposit);
        }

        double GetAllAreaScore(FlatInfo info) {
            return all_area_multiplier * info.total_area;
        }

        double GetCostScore(FlatInfo info, int max_cost) {
            if (info.price == 0 || info.price > max_cost) {
                return -cost_multiplier * info.deposit;
            } else {
                return -cost_multiplier * info.price;
            }
        }

        double GetLiveAreaScore(FlatInfo info) {
            if (std::abs(info.living_area) < eps) {
                return living_area_multiplier * living_area_part * info.total_area;
            } else {
                return living_area_multiplier * info.living_area;
            }
        }

        double GetKitchenAreaScore(FlatInfo info) {
            if (std::abs(info.kitchen_area) < eps || std::abs(info.kitchen_area) > max_kitchen_area) {
                return kitchen_area_multiplier * default_kitchen_area;
            } else {
                return kitchen_area_multiplier * info.kitchen_area;
            }
        }

        double GetComissionScore(FlatInfo info) {
            return -comission_multiplier * double(info.comission) * double(info.price);
        }

        double GetPrepaidMonthsScore(FlatInfo info) {
            return -prepaid_months_multiplier * double(info.deposit_months);
        }

        double GetChildrenScore(FlatInfo info) {
            return (info.children_allowed ? children_multiplier : 0.0);
        }

        double GetPetsScore(FlatInfo info) {
            return (info.pets_allowed ? pets_multiplier : 0.0);
        }

        double GetDishwasherScore(FlatInfo info) {
            return (info.has_dishwasher ? dishwasher_multiplier : 0.0);
        }

        double GetConditionerScore(FlatInfo info) {
            return (info.has_conditioner ? conditioner_multiplier : 0.0);
        }

        double GetApartmentsScore(FlatInfo info) {
            return (info.is_apartments ? apartments_multiplier : 0.0);
        }

        double GetRoomsScore(FlatInfo info) {
            if (info.room_number <= 1) {
                return 0.0;
            }
            if (info.room_number == 2) {
                return two_room_multiplier;
            }
            if (info.room_number == 3) {
                return three_room_multiplier;
            }
            if (info.room_number == 4) {
                return four_room_multiplier;
            }
            return four_room_multiplier + double(info.room_number) * additional_rooms_multiplier;
        }

        double GetWindowsViewScore(FlatInfo info) {
            if (info.windows_view.empty()) {
                return 0.0;
            }
            if (info.windows_view == "yard") {
                return windows_yard_multiplier;
            }
            if (info.windows_view == "street") {
                return windows_street_multiplier;
            }
            return windows_both_multiplier;
        }

        double GetRenovationScore(FlatInfo info) {
            if (info.renovation == "cosmetic") {
                return renovation_cosmetic_multiplier;
            }
            if (info.renovation == "euro") {
                return renovation_euro_multiplier;
            }
            if (info.renovation == "design") {
                return renovation_design_multiplier;
            }
            return 0.0;
        }

        double GetBathroomScore(FlatInfo info) {
            return std::max(bathroom_combined_multiplier * info.combined_bathroom_count, bathroom_separated_multiplier * info.separated_bathroom_count);
        }

        double GetBalconiesScore(FlatInfo info) {
            return std::max(balconies_balcony_multiplier * info.balcony_count, balconies_loggia_multiplier * info.loggia_count);
        }

        double GetUndergroundScore(FlatInfo info) {
            return underground_score_multiplier * (default_underground_score - info.underground_score);
        }

        int CalculateFlatScore(FlatInfo info, bool debugLogs) {
            double score = default_score_value;
            score += CalculateDepositScore(info);
            score += GetAllAreaScore(info);
            score += GetCostScore(info, max_possible_price);
            score += GetLiveAreaScore(info);
            score += GetKitchenAreaScore(info);
            score += GetComissionScore(info);
            score += GetPrepaidMonthsScore(info);
            
            
            score += GetChildrenScore(info);
            score += GetPetsScore(info);
            score += GetDishwasherScore(info);
            score += GetConditionerScore(info);
            score += GetApartmentsScore(info);
            score += GetRoomsScore(info);
            score += GetWindowsViewScore(info);
            score += GetRenovationScore(info);
            score += GetBathroomScore(info);
            score += GetBalconiesScore(info);
            score += GetUndergroundScore(info);
            if (debugLogs && false) {
                std::cerr << "CalculateDepositScore = " << CalculateDepositScore(info) << '\n';
                std::cerr << "GetAllAreaScore = " << GetAllAreaScore(info) << '\n';
                std::cerr << "GetCostScore = " << GetCostScore(info, max_possible_price) << '\n';
                std::cerr << "GetLiveAreaScore = " << GetLiveAreaScore(info) << '\n';
                std::cerr << "GetKitchenAreaScore = " << GetKitchenAreaScore(info) << '\n';
                std::cerr << "GetComissionScore = " << GetComissionScore(info) << '\n';
                std::cerr << "GetPrepaidMonthsScore = " << GetPrepaidMonthsScore(info) << '\n';
                std::cerr << "GetChildrenScore = " << GetChildrenScore(info) << '\n';
                std::cerr << "GetPetsScore = " << GetPetsScore(info) << '\n';
                std::cerr << "GetDishwasherScore = " << GetDishwasherScore(info) << '\n';
                std::cerr << "GetConditionerScore = " << GetConditionerScore(info) << '\n';
                std::cerr << "GetApartmentsScore = " << GetApartmentsScore(info) << '\n';
                std::cerr << "GetRoomsScore = " << GetRoomsScore(info) << '\n';
                std::cerr << "GetWindowsViewScore = " << GetWindowsViewScore(info) << '\n';
                std::cerr << "GetRenovationScore = " << GetRenovationScore(info) << '\n';
                std::cerr << "GetBathroomScore = " << GetBathroomScore(info) << '\n';
                std::cerr << "GetBalconiesScore = " << GetBalconiesScore(info) << '\n';
                std::cerr << "GetUndergroundScore = " << GetUndergroundScore(info) << '\n';
            }
            return int(score);
        }

    }
}