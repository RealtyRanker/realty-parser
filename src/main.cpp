#include <bits/stdc++.h>


#include <userver/clients/dns/component.hpp>
#include <userver/clients/http/component.hpp>
#include <userver/components/minimal_server_component_list.hpp>
#include <userver/server/handlers/ping.hpp>
#include <userver/server/handlers/server_monitor.hpp>
#include <userver/server/handlers/tests_control.hpp>
#include <userver/testsuite/testsuite_support.hpp>
#include <userver/utils/daemon_run.hpp>

#include "backgrounds/latest_flats_listener.hpp"

#include "components/s3api_client.hpp"
#include "components/telegram_bot.hpp"
#include "components/statistics.hpp"

#include "handlers/flats_history.hpp"
#include "handlers/hello.hpp"


int main(int argc, char* argv[]) {
  auto component_list = userver::components::MinimalServerComponentList()
                            .Append<userver::server::handlers::Ping>()
                            .Append<userver::server::handlers::ServerMonitor>()
                            .Append<userver::components::TestsuiteSupport>()
                            .Append<userver::components::HttpClient>()
                            .Append<realty-parser::components::Statistics>()
                            .Append<userver::server::handlers::TestsControl>()
                            .Append<userver::clients::dns::Component>()
                            .Append<realty-parser::components::S3ApiComponent>()
                            .Append<realty-parser::components::TelegramBotComponent>();

  realty-parser::AppendHelloHandler(component_list);
  realty-parser::AppendFlatsHistoryHandler(component_list);

  realty-parser::EnableLatestFlatsListener(component_list);

  std::cerr << "SERVICE STARTED\n";
  return userver::utils::DaemonMain(argc, argv, component_list);
}
