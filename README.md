# ldtester
<hr>

This is a utility for load testing, it can be used as a terminal tool or an http server.

## Installation
<hr>

```shell
go install github.com/tagirmukail/ldtester/cmd/ldtester@latest
```

## Usage
<hr>

#### Configuration
```yaml
LogLevel: 4 # 5: Debug, 4:Info, 3:Error
Server: # configuration for server
  Port: 8000 # listen port
  WriteTimeout: 430 # sec
  ReadTimeout: 430 # sec

LoadTest: # configuration for load testing
  StressTestTimeout: 30 # in sec # if the request didn't respond installed time, load testing will be stopped forcibly. 
  MaxIdleConnPerHost: 200 # the number of idle connections per host
  DisableCompression: false
  DisableKeepAlive: false
  UseHTTP2: false
  Timeout: 3 # sec # max allowed request time
  Method: "GET" # request method for all urls
  AcceptHeaderRequest: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
  UserAgent: "" # install your user agent
```

### Http Server

Use this command for start the server.
```shell
ldtester --config ${path_to_config} server
```

When the server will be started. Send POST Request by endpoint `/load`.
Configuration for load testing can be changed in `/load` request by headers or query params.

| config variable name | query param        | header                 |
|----------------------|--------------------|------------------------|
| MaxIdleConnPerHost   | tmaxidleconnhost   |   T-Max-Idle-Conn-Host |
| DisableCompression   | tdisablecompress   |   T-Disable-Compress   |
| DisableKeepAlive     | tdisablekeepalive  |   T-Disable-Keep-Alive |
| Timeout              | treqtimeout        |   T-Req-Timeout        |
| Method               | tmethod            |   T-Method             |
| AcceptHeaderRequest  |         -          |   T-Accept             |
| UserAgent            |         -          |   T-User-Agent         |

**_Example_**:
```shell
curl -X POST -H "T-Max-Idle-Conn-Host: 150" http://localhost:8080/load?treqtimeout=4
```

**_Output format_**:

```json
{
  "message": "",
  "load_test_config": {
    "concurrency_num_per_item": 200,
    "requests_num": 3,
    "max_idle_conn_per_host": 200,
    "disable_compression": false,
    "disable_keep_alive": false,
    "use_http_2": false,
    "timeout": 3000000000,
    "method": "GET",
    "accept_header_request": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
    "user_agent": "...."
  },
  "data": {
    "https://www.test.com/query1": {
      "recommend_req_count": 2000,
      "total_req_count": 2500,
      "err_request_count": 200,
      "max_req_time": 2.34,
      "slow_req_count": 300
    },
    "https://www.yandex.com/query2": {
      "recommend_req_count": 2500,
      "total_req_count": 2700,
      "err_request_count": 100,
      "max_req_time": 2.02,
      "slow_req_count": 100
    }
  }
}
```

### Terminal tool

Use with urls csv file.
```shell
ldtester --config ${path_to_config} load -f ${path_to_csv_file}
```

Use with one url.
```shell
ldtester load -u "https://www.test.com/some/query" -m GET
```

csv data format:
```
url
https://www.test.com/some/query
https://www.yandex.com/query
...
```

For `load` command can be used flags:
```
--loadcsv -f csv file with urls.
--url -u one url for load testing.
--method -m request method for load testing.
```

## Build and run the docker image

### Build image
```shell
docker build -t ldtester .
```

### Run like a server
```shell
docker run -d --name ldtester -p 8000:8000 ldtester /app/ldtester server
```

### Run like a terminal tool
```shell
docker run --name ldtester ldtester /app/ldtester load -f ${path_to_csv_file}
```