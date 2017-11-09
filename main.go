package main

import (
	"net/http"
	"encoding/json"
	"io"
	"io/ioutil"
	"errors"
	"database/sql"
	"fmt"
	"time"
	"flag"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var config Settings

const metricPath =  "/metrics"
const interval = time.Minute * 5

//struct for queries.json file
type Query struct {
	Name	   string `json:"name"` //name of metric
	SQL        string `json:"sql"` //query
}

type QueryList []Query

//connection string and queries
type Settings struct {
	ServicePort string `json:"service_port"`   //httpHandler port
	ConStr      string `json:"conStr"`         //connection string
	QueriesFile string `json:"queries_file"`   //file with queries

}

type QueryResult struct {
	Query  *Query
	Result map[string]prometheus.Gauge
}

func getQueries(r io.Reader) (QueryList, error) {
	queries := make(QueryList, 0)
	parsedQueries := make([]Query, 0)

	b, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, &parsedQueries); err != nil {
		glog.Fatalf("Can't parse json: ", err)
	}

	for _, q := range parsedQueries {
			if q.SQL == "" {
				return nil, errors.New("SQL statement required")
			}

			queries = append(queries, q)
	}
	return queries, nil
}

func init() {
	config = getConfig()
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func getConfig() Settings {
	config := Settings{}
	configFile, err := os.Open("config.json")
	defer configFile.Close()
	if err != nil {
		glog.Fatalln("Can't open config.json: ", err)
	}
	if err = json.NewDecoder(configFile).Decode(&config); err != nil {
		glog.Fatalln("Decode config.json error: ", err)
	}
	return config
}

func exporting() {
	db := conMssql()
	defer db.Close()
	glog.Infoln("Enter to exporting function")
	file, err := os.Open(config.QueriesFile)
	defer file.Close()
	if err!=nil {
		glog.Fatalf("Queries not found: ", err)
	}

	queries, err := getQueries(file)
	if err!=nil {
		glog.Fatalf("Error with parse json: ", err)
	}

	x := make(map[string]prometheus.Gauge) //map where key is name of sql query and result is prometheus gauge

	for {
		select {
		case <-time.After(interval):
			for _, q := range queries {
				findRow(db, q, x)
			}
		}
	}
}

func findRow(db *sql.DB, q Query, x map[string]prometheus.Gauge) {
	query, err := db.Query(q.SQL)
	if err != nil {
		glog.Errorln("Query: ", query, " Error: ", err)
	}
	defer query.Close()
	for query.Next() {
		select {
		case <-time.After(time.Second*1):
			var result int
			if err := query.Scan(&result); err != nil {
				glog.Errorln("Can't find result: ", err)
			}

			if _, ok := x[q.Name]; ok { // Metric with this name is already registered
				x[q.Name].Set(float64(result))
				break
			}
			x[q.Name] = prometheus.NewGauge(prometheus.GaugeOpts{
				Name: fmt.Sprintf("query_result_%s", q.Name),
				Help: "SQL query result",
			})
			prometheus.MustRegister(x[q.Name])
			x[q.Name].Set(float64(result))
		}
	}
}

func conMssql() *sql.DB {
	glog.Infoln("Connect to MSSQL")
	db, err := sql.Open("mssql", config.ConStr)
	if err != nil {
		glog.Errorln("Connection Error:", err)
		time.Sleep(time.Second * 5)
	}
	return db
}

func main() {
	for {
		go exporting()
		glog.Infoln("Start prometheus http server on port ", config.ServicePort)
		http.Handle(metricPath, promhttp.Handler())
		if err := http.ListenAndServe(config.ServicePort, nil); err != nil {
			glog.Errorln(err)
		}
	}
	glog.Fatalf("Unexpected error")
}
