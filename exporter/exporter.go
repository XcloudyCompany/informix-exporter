package exporter

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	conf "github.com/anmartsan/informix-exporter/config"
	informix "github.com/anmartsan/informix-exporter/sql"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	profileMetrics = map[string]metric{
		"pf_isamtot":     metric{Name: "pf_isamtot", Help: "Total ISAM"},
		"pf_isopens":     metric{Name: "pf_isopens", Help: "Total ISAM opens"},
		"pf_isreads":     metric{Name: "pf_isreads", Help: "Total ISAM reads"},
		"pf_iswrites":    metric{Name: "pf_iswrites", Help: "Total ISAM writes"},
		"pf_isrewrites":  metric{Name: "pf_isrewrites", Help: "Total ISAM updates"},
		"pf_isdeletes":   metric{Name: "pf_isdeletes", Help: "Total ISAM deletes"},
		"pf_iscommits":   metric{Name: "pf_iscommits", Help: "Total commits"},
		"pf_isrollbacks": metric{Name: "pf_isrollbacks", Help: "Total rollbacks"},
		"pf_latchwts":    metric{Name: "pf_latchwts", Help: "Total latch waits"},
		"pf_buffwts":     metric{Name: "pf_buffwts", Help: "Total buffer waits"},
		"pf_lockreqs":    metric{Name: "pf_lockreqs", Help: "Total lock request"},
		"pf_lockwts":     metric{Name: "pf_lockwts", Help: "Total locks waits"},
		"pf_ckptwts":     metric{Name: "pf_ckptwts", Help: "Total checkpoint waits"},
		"pf_plgwrites":   metric{Name: "pf_plgwrites", Help: "Total physical log writes"},
		"pf_pagreads":    metric{Name: "pf_pagreads", Help: "Total page reads"},
		"pf_btradata":    metric{Name: "pf_btradata", Help: "Total pf_btradata"},
		"pf_rapgs_used":  metric{Name: "pf_rapgs_used", Help: "Total pf_rapgs_used"},
		"pf_btraidx":     metric{Name: "btraidx", Help: "Read Ahead Index"},
		"pf_dpra":        metric{Name: "dpra", Help: "dpra"},
		"pf_seqscans":    metric{Name: "pf_seqscans", Help: "Total secuencial scans"},
		"pagreads_2K":    metric{Name: "pagreads_2K", Help: "Total paginas leidas 2k"},
		"bufreads_2K":    metric{Name: "bufreads_2K", Help: "Total buffer reads 2k"},
		"pagwrites_2K":   metric{Name: "pagwrites_2K", Help: "Total page writes 2k "},
		"bufwrites_2K":   metric{Name: "bufwrites_2K", Help: "Total buffer writes 2k"},
		"bufwaits_2K":    metric{Name: "bufwaits_2K", Help: "Total buffer waits 2k"},
		"pagreads_16K":   metric{Name: "pagreads_16K", Help: "Total page reads 16k"},
		"bufreads_16K":   metric{Name: "bufreads_16K", Help: "Total buffer reads 16k"},
		"pagwrites_16K":  metric{Name: "pagwrites_16K", Help: "Total page writes 16k"},
		"bufwrites_16K":  metric{Name: "bufwrites_16K", Help: "Total buffer writes 16k"},
		"bufwaits_16K":   metric{Name: "bufwaits_16K", Help: "Total buffer waits 16k"},
		"net_connects":   metric{Name: "net_connects", Help: "Number of connects"},
		"pf_totalsorts":  metric{Name: "pf_totalsorts", Help: "Total Sorts"},
		"pf_memsorts":    metric{Name: "pf_memsorts", Help: "Mem sorts"},
		"pf_disksorts":   metric{Name: "pf_disksorts", Help: "Disk sorts"},
	}
	chunkMetrics = map[string]metric{
		"reads":     metric{Name: "reads", Help: "Chunk reads"},
		"writes":    metric{Name: "writes", Help: "Chunk writes"},
		"readtime":  metric{Name: "readtime", Help: "Chunk read time"},
		"writetime": metric{Name: "writetime", Help: "Chunk write time"},
	}
	dbsMetrics = map[string]metric{
		"freespace": metric{Name: "freespace", Help: "dbs free space"},
	}
)

type metric struct {
	Name string
	Help string
}

// ProfileMetrics estructura
type ProfileMetrics struct {
	Metrics       map[string]*prometheus.GaugeVec
	MetricsChunks map[string]*prometheus.GaugeVec
	MetricsDbs    map[string]*prometheus.GaugeVec

	Configuracion conf.ConfigYaml
	Servers       conf.InstanceList
}

// NewExporter represents a request to run a command.
func NewExporter(probes *conf.ConfigYaml, instancias *conf.InstanceList) *ProfileMetrics {

	e := ProfileMetrics{Metrics: map[string]*prometheus.GaugeVec{}, MetricsChunks: map[string]*prometheus.GaugeVec{}, MetricsDbs: map[string]*prometheus.GaugeVec{}}
	e.Configuracion.Metrics = make([]conf.Probes, len(probes.Metrics))

	e.Servers = *instancias

	//Custom querys
	for key, value := range probes.Metrics {
		e.Metrics[value.Parametro] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "informix",
			Name:      probes.Metrics[key].Parametro,
			Help:      probes.Metrics[key].Description},
			[]string{"informixserver", probes.Metrics[key].Label})

		e.Configuracion.Metrics[key] = value

	}

	//Automatic Querys

	for key := range profileMetrics {
		e.Metrics[key] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "informix",
			Name:      key,
			Help:      key},
			[]string{"informixserver", "automatic"})
	}

	//Automatic chunks Querys

	for key := range chunkMetrics {

		e.MetricsChunks[key] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "informix",
			Name:      key,
			Help:      key},
			[]string{"informixserver", "chunk", "metrica", "automatic"})

	}
	//Automatic dbs Querys

	for key := range dbsMetrics {

		e.MetricsDbs[key] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "informix",
			Name:      key,
			Help:      key},
			[]string{"informixserver", "dbs", "metrica", "automatic"})

	}

	fmt.Println(e.MetricsDbs)

	return &e
}

// Describe function
func (p *ProfileMetrics) Describe(ch chan<- *prometheus.Desc) {

	for _, m := range p.Metrics {
		m.Describe(ch)
	}
	for _, m := range p.MetricsChunks {
		m.Describe(ch)
	}
	for _, m := range p.MetricsDbs {
		m.Describe(ch)
	}
}

// Collect function
func (p *ProfileMetrics) Collect(ch chan<- prometheus.Metric) {

	p.scrape(ch)
	for _, m := range p.Metrics {

		m.Collect(ch)
	}
	for _, m := range p.MetricsChunks {

		m.Collect(ch)
	}
	for _, m := range p.MetricsDbs {

		m.Collect(ch)
	}

}

func customQuerys(p *ProfileMetrics, database *sql.DB, servidor conf.Instance) {

	var data float64

	for _, value := range p.Configuracion.Metrics {

		rows, err := informix.QueryDatabase(database, value.Query)
		if err != nil {
			log.Println("Error in  Query custom: \n", err)
			continue
		}
		defer rows.Close()

		for rows.Next() {

			err := rows.Scan(&data)
			if err != nil {
				fmt.Println("Error en scan")
			}
			p.Metrics[value.Parametro].WithLabelValues(servidor.Informixserver, value.Label).Set(data)

		}

		rows.Close()

	}

}

func automaticQuerys(p *ProfileMetrics, database *sql.DB, servidor conf.Instance) {

	var (
		name  string
		value float64
		err   error
	)
	rows, err := informix.QueryDatabase(database, "select name,value from sysshmhdr")

	if err != nil {
		log.Println("Error in  Query sysshmhdr: \n", err)

	}
	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&name, &value)

		if err != nil {
			log.Println("Error in Scan", err)

		}
		if _, ok := p.Metrics[strings.TrimSpace(name)]; ok {
			p.Metrics[strings.TrimSpace(name)].WithLabelValues(servidor.Informixserver, "automatico").Set(value)

		}

	}

	rows.Close()

}

func automaticQuerysChunks(p *ProfileMetrics, database *sql.DB, servidor conf.Instance) {

	var (
		fname        string
		pagesread    int64
		pageswritten int64
		readtime     float64
		writetime    float64
	)
	rows, err := informix.QueryDatabase(database, "select fname,pagesread,pageswritten,readtime,writetime from syschktab ")

	if err != nil {
		log.Fatal("Error en Query: \n", err)
	}
	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&fname, &pagesread, &pageswritten, &readtime, &writetime)
		if err != nil {
			log.Fatal("Error en Scan", err)
		}

		p.MetricsChunks["reads"].WithLabelValues(servidor.Informixserver, fname, "reads", "automatic").Set(float64(pagesread))
		p.MetricsChunks["writes"].WithLabelValues(servidor.Informixserver, fname, "writes", "automatic").Set(float64(pageswritten))
		p.MetricsChunks["readtime"].WithLabelValues(servidor.Informixserver, fname, "readtime", "automatic").Set(float64(readtime))
		p.MetricsChunks["writetime"].WithLabelValues(servidor.Informixserver, fname, "writetime", "automatic").Set(float64(writetime))

	}
	rows.Close()

}

func automaticQuerysDbs(p *ProfileMetrics, database *sql.DB, servidor conf.Instance) {

	var (
		fname   string
		freedbs float64
	)
	rows, err := informix.QueryDatabase(database, `select
	dbs.name[1,20] dbspace, 
	round(SUM(chk.nfree)*2/1024,2) MBlibres
	
	from sysdbspaces dbs, syschunks chk
	where dbs.dbsnum=chk.dbsnum and is_sbchunk <> 1
	group by dbs.name
	UNION
	select
	dbs.name[1,20] dbspace, 
	round(SUM(chk.udfree)*2/1024,2) MBlibres
	
	from sysdbspaces dbs, syschunks chk
	where dbs.dbsnum=chk.dbsnum and is_sbchunk=1
	group by dbs.name
	order by 1;
	`)
	if err != nil {
		log.Println("Error en Query: \n", err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&fname, &freedbs)

		if err != nil {
			log.Println("Error en Scan", err)
		}
		//p.MetricsDbs["freespace"].WithLabelValues(servidor.Informixserver, "rootdbs", "reads", "automatic").Set(float64(200))
		p.MetricsDbs["freespace"].WithLabelValues(servidor.Informixserver, fname, "freespace", "automatic").Set(freedbs)
	}
	fmt.Println("aqui")
	rows.Close()

}

func (p *ProfileMetrics) scrape(ch chan<- prometheus.Metric) error {

	for _, servidor := range p.Servers.Servers {

		a := "DSN=" + servidor.Informixserver

		database := informix.OpenDatabase(a)
		err := database.Ping()

		if err != nil {
			fmt.Println(err)
			continue
		}
		customQuerys(p, database, servidor)
		automaticQuerys(p, database, servidor)
		automaticQuerysChunks(p, database, servidor)
		automaticQuerysDbs(p, database, servidor)

		informix.CloseDatabase(database)

	}
	return nil
}
