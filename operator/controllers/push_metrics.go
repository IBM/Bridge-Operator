package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	LSF_POD     = "lsf"
	SLURM_POD   = "slurm"
	RAY_POD     = "ray"
	QUANTUM_POD = "quantum"
	UNKNOWN_POD = "unknown"

	POD_CREATED      = "created"
	POD_FAILED       = "failed"
	POD_KILLED       = "killed"
	POD_JOBFAILED    = "jobfailed"
	POD_JOBCOMPLETED = "jobcompleted"
)

var (
	podscreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pods_created_total",
			Help: "Number of created pods",
		},
	)
	lsf_podscreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lsf_pods_created_total",
			Help: "Number of created lsf pods",
		},
	)
	slurm_podscreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "slurm_pods_created_total",
			Help: "Number of created slurm pods",
		},
	)
	ray_podscreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ray_pods_created_total",
			Help: "Number of created ray pods",
		},
	)
	quantum_podscreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quantum_pods_created_total",
			Help: "Number of created quantum pods",
		},
	)
	podsfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pods_failed_total",
			Help: "Number of failed pods",
		},
	)
	lsf_podsfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lsf_pods_failed_total",
			Help: "Number of failed lsf pods",
		},
	)
	slurm_podsfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "slurm_pods_failed_total",
			Help: "Number of failed slurm pods",
		},
	)
	ray_podsfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ray_pods_failed_total",
			Help: "Number of failed ray pods",
		},
	)
	quantum_podsfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quantum_pods_failed_total",
			Help: "Number of failed quantum pods",
		},
	)
	podskilled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pods_killed_total",
			Help: "Number of killed pods",
		},
	)
	lsf_podskilled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lsf_pods_killed_total",
			Help: "Number of killed lsf pods",
		},
	)
	slurm_podskilled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "slurm_pods_killed_total",
			Help: "Number of killed slurm pods",
		},
	)
	ray_podskilled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ray_pods_killed_total",
			Help: "Number of killed ray pods",
		},
	)
	quantum_podskilled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quantum_pods_killed_total",
			Help: "Number of killed quantum pods",
		},
	)
	podsjobfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pods_job_failed_total",
			Help: "Number of failed remote jobs",
		},
	)
	lsf_podsjobfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lsf_pods_job_failed_total",
			Help: "Number of failed lsf remote jobs",
		},
	)
	slurm_podsjobfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "slurm_pods_job_failed_total",
			Help: "Number of failed remote slurm jobs",
		},
	)
	ray_podsjobfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ray_pods_job_failed_total",
			Help: "Number of failed remote ray jobs",
		},
	)
	quantum_podsjobfailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quantum_pods_job_failed_total",
			Help: "Number of failed remote quantum jobs",
		},
	)
	podsjobcompleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pods_job_completed_total",
			Help: "Number of completed remote jobs",
		},
	)
	lsf_podsjobcompleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lsf_pods_job_completed_total",
			Help: "Number of completed lsf remote jobs",
		},
	)
	slurm_podsjobcompleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "slurm_pods_job_completed_total",
			Help: "Number of completed remote slurm jobs",
		},
	)
	ray_podsjobcompleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ray_pods_job_completed_total",
			Help: "Number of completed remote ray jobs",
		},
	)
	quantum_podsjobcompleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quantum_pods_job_completed_total",
			Help: "Number of completed remote quantum jobs",
		},
	)
	podsjobduration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pods_job_duration_total",
			Help: "Overall duration of remote jobs in mins",
		},
	)
	lsf_podsjobduration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "lsf_pods_job_duration_total",
			Help: "Overall duration of lsf remote jobs in mins",
		},
	)
	slurm_podsjobduration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "slurm_pods_job_duration_total",
			Help: "Overall duration of remote slurm jobs in mins",
		},
	)
	ray_podsjobduration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ray_pods_job_duration_total",
			Help: "Overall duration of remote ray jobs in mins",
		},
	)
	quantum_podsjobduration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "quantum_pods_job_duration_total",
			Help: "Overall duration of remote quantum jobs in mins",
		},
	)

	lsf_podcounters = map[string]prometheus.Counter{
		POD_CREATED:      lsf_podscreated,
		POD_FAILED:       lsf_podsfailed,
		POD_KILLED:       lsf_podskilled,
		POD_JOBFAILED:    lsf_podsjobfailed,
		POD_JOBCOMPLETED: lsf_podsjobcompleted,
	}
	slurm_podcounters = map[string]prometheus.Counter{
		POD_CREATED:      slurm_podscreated,
		POD_FAILED:       slurm_podsfailed,
		POD_KILLED:       slurm_podskilled,
		POD_JOBFAILED:    slurm_podsjobfailed,
		POD_JOBCOMPLETED: slurm_podsjobcompleted,
	}
	ray_podcounters = map[string]prometheus.Counter{
		POD_CREATED:      ray_podscreated,
		POD_FAILED:       ray_podsfailed,
		POD_KILLED:       ray_podskilled,
		POD_JOBFAILED:    ray_podsjobfailed,
		POD_JOBCOMPLETED: ray_podsjobcompleted,
	}
	quantum_podcounters = map[string]prometheus.Counter{
		POD_CREATED:      quantum_podscreated,
		POD_FAILED:       quantum_podsfailed,
		POD_KILLED:       quantum_podskilled,
		POD_JOBFAILED:    quantum_podsjobfailed,
		POD_JOBCOMPLETED: quantum_podsjobcompleted,
	}

	counters = map[string]map[string]prometheus.Counter{
		LSF_POD:     lsf_podcounters,
		SLURM_POD:   slurm_podcounters,
		RAY_POD:     ray_podcounters,
		QUANTUM_POD: quantum_podcounters,
	}

	gauges = map[string]prometheus.Gauge{
		LSF_POD:     lsf_podsjobduration,
		SLURM_POD:   slurm_podsjobduration,
		RAY_POD:     ray_podsjobduration,
		QUANTUM_POD: quantum_podsjobduration,
	}
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		podscreated, lsf_podscreated, slurm_podscreated, ray_podscreated, quantum_podscreated,
		podsfailed, lsf_podsfailed, slurm_podsfailed, ray_podsfailed, quantum_podsfailed,
		podskilled, lsf_podskilled, slurm_podskilled, ray_podskilled, quantum_podskilled,
		podsjobfailed, lsf_podsjobfailed, slurm_podsjobfailed, ray_podsjobfailed, quantum_podsjobfailed,
		podsjobcompleted, lsf_podsjobcompleted, slurm_podsjobcompleted, ray_podsjobcompleted, quantum_podsjobcompleted,
		podsjobduration, lsf_podsjobduration, slurm_podsjobduration, ray_podsjobduration, quantum_podsjobduration,
	)
}
