# Metis Sequencer Monitoring Setup Example

## Edit your [./docker-compose.yml](./docker-compose.yml) to setup your promethues stacks

**You can skip this step if you deploy the services with your sequencer node on the same instance**

```yaml
services:
  metis-sequencer-exporter:
    image: ghcr.io/metisprotocol/metis-sequencer-exporter:main
    pull_policy: always
    ports:
      - 21012:21012
    extra_hosts:
      - "host.docker.internal:host-gateway"
    command:
      - -url.state.seq
      - http://you-sequencer-ip:9545/health
      # your rest rpc service
      - -url.state.node
      - http://you-sequencer-ip:1317/metis/latest-span
      # your dtl service
      - -url.state.l1dtl
      - http://you-sequencer-ip:7878/eth/context/latest
```

## Edit [./config/prometheus.yml](./config/prometheus.yml) file to setup your prometheus configuration

**You can skip this step if you don't have any custom configurations**

```yaml
alerting:
  alertmanagers:
    - static_configs:
        - targets: ["alertmanager:9093"]

rule_files:
  - "rules/*.yml"

scrape_configs:
  - job_name: "metis-sequencer-exporter"
    scrape_interval: 5s
    static_configs:
      - targets: ["metis-sequencer-exporter:21012"]
  - job_name: "node-exporter"
    static_configs:
      - targets: ["node-exporter:9100"]
```

## Edit the [./config/prometheus-web.yml](./config/prometheus-web.yml) to add basic auth for for your prometheus

**You can skip this step if you don't need to add your own user**

```yml
basic_auth_users:
  # Usernames and hashed passwords that have full access to the web server via basic authentication.
  metis: "$2a$12$RpMALcEU6ycfhsM.h.JrD.CBLeqnCpdQArPn5vuVQe2oGQtoXQ7em"
```

the metis user is reserved for Metis Devops team, Don't remove it!

btw, if you want add your own user, you can use https://bcrypt-generator.com to generate the password

## Setup your AlertManager configuration

There is an example to send the alerts to your telegram group

1. Add `@BotFather` to your telemgram contacts
2. Input `/newbot` command and then flow the instructions, you will get the `bot_token` in a red text
3. Create a group and add your bot to the group
4. Refer to this StackOverflow [answer](https://stackoverflow.com/questions/32423837/telegram-bot-how-to-get-a-group-chat-id) to get your group `chat_id`
5. Use the following shell to create the configuration file

```sh
# You MUST update these variables first !!
TG_BOT_TOKEN="YOUR BOT TOKEN!!"
TG_CHAT_ID="YOUR CHAT ID!!"

cat << EOF > ./config/alertmanager.yml
route:
  receiver: telegram

receivers:
  - name: "telegram"
    telegram_configs:
      - send_resolved: true
        api_url: https://api.telegram.org
        bot_token: "$TG_BOT_TOKEN"
        chat_id: "$TG_CHAT_ID"
        parse_mode: ""
EOF
```

## Add testing alert rule to ensure your settting correct

```yaml
cat << EOF > ./config/rules/testing.yml
groups:
  - name: testing
    rules:
      - alert: Your alert works!
        expr: vector(1)
EOF
```

## Start

```sh
docker compose up -d
```

If you receive the `Your alert works!` alert

you can remove the testing rule file and reload the promethues files

```sh
rm ./config/rules/testing.yml
docker compose exec prometheus kill -SIGHUP 1
```

If you receive the `ScrapeFailures` alert

please check out the exporter log and get the reason.

```sh
docker compose logs --tail=100 metis-sequencer-exporter
```

## Update security group and firewall configuration

We use prometheus federation api to fetch the metrics from your instance

Please add these rule to your security group and firewall config

| Port | Description       | Protocol | Source       |
| ---- | ----------------- | -------- | ------------ |
| 9090 | Promethues server | TCP      | 3.89.0.69/32 |
