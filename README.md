# sqlExporter
Very simple sql query result exporter. Only for MSSQL, another DBs is coming soon!

So, this is not a real app, actually, it's just a script for sysadmins, who working with Prometheus.

Need's go 1.5 ot higher.

## Getting Started

Just write into the queries.json file your quieries and build app.

queries.json (or another file, look at config.json) must contain a json array!
### queries.json example
```json
[
  {
    "name": "my_clients",
    "sql": "select count(*) from [my_db].[dbo].[my_clients] where [] = 0 and ([live_status] = 0 or ([live_status] = 3 and [loginTries] < 5))"
  },
  {
    "name": "my_new_24h_dynamic",
    "sql": "SELECT count(*) FROM [my_db].[dbo].[my_clients] where datediff(hour,[createDate],getDate()) < 24 and buckettype = '0'"
  },
  {
    "name": "my_dead_clients",
    "sql": "select count(*) from [my_db].[dbo].[my_store] where [client] = 0 and [live_status] = 3 and [loginTries] = 5"
  }
]
```
PostgreSQL and MySQL is coming soon
