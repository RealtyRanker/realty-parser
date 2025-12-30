#pragma once

#include <bits/stdc++.h>

namespace realty-parser {
    struct FlatSearchFilter {
        int min_price = 0;
        int max_price = 0;
        int min_area = 0;
        int max_area = 0;
        std::string sort = "";
        int room1 = 0;
        int room2 = 0;
        int room3 = 0;
        int room4 = 0;
        int room5 = 0;
        int room6 = 0;
        int page = 1;
        int limit = 100;
    };

    void AddParamToUrl(const std::string& name, int value, std::string& url) {
        if (value != 0) {
            url += "&" + name + "=" + std::to_string(value);
        }
    }

    std::string GetFlatSearchUrl(const FlatSearchFilter& filter) {
        std::string url = "https://www.cian.ru/cat.php?deal_type=rent&engine_version=2&offer_type=flat&region=1&type=4";
        AddParamToUrl("minprice", filter.min_price, url);
        AddParamToUrl("maxprice", filter.max_price, url);
        AddParamToUrl("minarea", filter.min_area, url);
        AddParamToUrl("maxarea", filter.max_area, url);
        if (!filter.sort.empty()) {
            url += "&sort=" + filter.sort;
        }
        AddParamToUrl("room1", filter.room1, url);
        AddParamToUrl("room2", filter.room2, url);
        AddParamToUrl("room3", filter.room3, url);
        AddParamToUrl("room4", filter.room4, url);
        AddParamToUrl("room5", filter.room5, url);
        AddParamToUrl("room6", filter.room6, url);
        AddParamToUrl("limit", filter.limit, url);
        AddParamToUrl("p", filter.page, url);
        return url;
    }
}