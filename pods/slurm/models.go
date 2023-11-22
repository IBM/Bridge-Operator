package main

// https://slurm.schedmd.com/rest_api.html#v0.0.39_job_submission

type JobSubmissionBody struct {
	Script string `json:"script,omitempty"`
	Job    Job    `json:"job,omitempty"`
	Jobs   []Job  `json:"jobs,omitempty"`
}

// https://slurm.schedmd.com/rest_api.html#v0.0.39_job_desc_msg

type Job struct {
	Account                 *string           `json:"account,omitempty"`
	AccountGatherFrequency  *string           `json:"account_gather_frequency,omitempty"`
	AdminComment            *string           `json:"admin_comment,omitempty"`
	AllocationNodeList      *string           `json:"allocation_node_list,omitempty"`
	AllocationNodePort      *int32            `json:"allocation_node_port,omitempty"`
	Argv                    *[]string         `json:"argv,omitempty"`
	Array                   *string           `json:"array,omitempty"`
	BatchFeatures           *string           `json:"batch_features,omitempty"`
	BeginTime               *int64            `json:"begin_time,omitempty"`
	Flags                   *[]string         `json:"flags,omitempty"`
	BurstBuffer             *string           `json:"burst_buffer,omitempty"`
	Clusters                *string           `json:"clusters,omitempty"`
	ClusterConstraint       *string           `json:"cluster_constraint,omitempty"`
	Comment                 *string           `json:"comment,omitempty"`
	Contiguous              *bool             `json:"contiguous,omitempty"`
	Container               *string           `json:"container,omitempty"`
	ContainerID             *string           `json:"container_id,omitempty"`
	CoreSpecification       *int32            `json:"core_specification,omitempty"`
	ThreadSpecification     *int32            `json:"thread_specification,omitempty"`
	CPUBinding              *string           `json:"cpu_binding,omitempty"`
	CPUBindingFlags         *[]string         `json:"cpu_binding_flags,omitempty"`
	CPUFrequency            *string           `json:"cpu_frequency,omitempty"`
	CPUsPerTres             *string           `json:"cpus_per_tres,omitempty"`
	Crontab                 *string           `json:"crontab,omitempty"`
	Deadline                *int64            `json:"deadline,omitempty"`
	DelayBoot               *int32            `json:"delay_boot,omitempty"`
	Dependency              *string           `json:"dependency,omitempty"`
	EndTime                 *int64            `json:"end_time,omitempty"`
	Environment             map[string]string `json:"environment"`
	ExcludedNodes           *[]string         `json:"excluded_nodes,omitempty"`
	Extra                   *string           `json:"extra,omitempty"`
	Constraints             *string           `json:"constraints,omitempty"`
	GroupID                 *string           `json:"group_id,omitempty"`
	HetjobGroup             *int32            `json:"hetjob_group,omitempty"`
	Immediate               *bool             `json:"immediate,omitempty"`
	JobID                   *int32            `json:"job_id,omitempty"`
	KillOnNodeFail          *bool             `json:"kill_on_node_fail,omitempty"`
	Licenses                *string           `json:"licenses,omitempty"`
	MailType                *[]string         `json:"mail_type,omitempty"`
	MailUser                *string           `json:"mail_user,omitempty"`
	MCSLabel                *string           `json:"mcs_label,omitempty"`
	MemoryBinding           *string           `json:"memory_binding,omitempty"`
	MemoryBindingType       *[]string         `json:"memory_binding_type,omitempty"`
	MemoryPerTres           *string           `json:"memory_per_tres,omitempty"`
	Name                    *string           `json:"name,omitempty"`
	Network                 *string           `json:"network,omitempty"`
	Nice                    *int32            `json:"nice,omitempty"`
	Tasks                   *int32            `json:"tasks,omitempty"`
	OpenMode                *[]string         `json:"open_mode,omitempty"`
	ReservePorts            *int32            `json:"reserve_ports,omitempty"`
	Overcommit              *bool             `json:"overcommit,omitempty"`
	Partition               *string           `json:"partition,omitempty"`
	DistributionPlaneSize   *int32            `json:"distribution_plane_size,omitempty"`
	PowerFlags              *[]string         `json:"power_flags,omitempty"`
	Prefer                  *string           `json:"prefer,omitempty"`
	Hold                    *bool             `json:"hold,omitempty"`
	Priority                *uint32           `json:"priority,omitempty"`
	Profile                 *[]string         `json:"profile,omitempty"`
	QoS                     *string           `json:"qos,omitempty"`
	Reboot                  *bool             `json:"reboot,omitempty"`
	RequiredNodes           *[]string         `json:"required_nodes,omitempty"`
	Requeue                 *bool             `json:"requeue,omitempty"`
	Reservation             *string           `json:"reservation,omitempty"`
	Script                  *string           `json:"script,omitempty"`
	Shared                  *[]string         `json:"shared,omitempty"`
	Exclusive               *[]string         `json:"exclusive,omitempty"`
	Oversubscribe           *bool             `json:"oversubscribe,omitempty"`
	SiteFactor              *int32            `json:"site_factor,omitempty"`
	SpankEnvironment        *[]string         `json:"spank_environment,omitempty"`
	Distribution            *string           `json:"distribution,omitempty"`
	TimeLimit               *uint32           `json:"time_limit,omitempty"`
	TimeMinimum             *uint32           `json:"time_minimum,omitempty"`
	TresBind                *string           `json:"tres_bind,omitempty"`
	TresFreq                *string           `json:"tres_freq,omitempty"`
	TresPerJob              *string           `json:"tres_per_job,omitempty"`
	TresPerNode             *string           `json:"tres_per_node,omitempty"`
	TresPerSocket           *string           `json:"tres_per_socket,omitempty"`
	TresPerTask             *string           `json:"tres_per_task,omitempty"`
	UserID                  *string           `json:"user_id,omitempty"`
	WaitAllNodes            *bool             `json:"wait_all_nodes,omitempty"`
	KillWarningFlags        *[]string         `json:"kill_warning_flags,omitempty"`
	KillWarningSignal       *string           `json:"kill_warning_signal,omitempty"`
	KillWarningDelay        *uint16           `json:"kill_warning_delay,omitempty"`
	CurrentWorkingDirectory *string           `json:"current_working_directory,omitempty"`
	CPUsPerTask             *int32            `json:"cpus_per_task,omitempty"`
	MinimumCPUs             *int32            `json:"minimum_cpus,omitempty"`
	MaximumCPUs             *int32            `json:"maximum_cpus,omitempty"`
	Nodes                   *string           `json:"nodes,omitempty"`
	MinimumNodes            *int32            `json:"minimum_nodes,omitempty"`
	MaximumNodes            *int32            `json:"maximum_nodes,omitempty"`
	MinimumBoardsPerNode    *int32            `json:"minimum_boards_per_node,omitempty"`
	MinimumSocketsPerBoard  *int32            `json:"minimum_sockets_per_board,omitempty"`
	SocketsPerNode          *int32            `json:"sockets_per_node,omitempty"`
	ThreadsPerCore          *int32            `json:"threads_per_core,omitempty"`
	TasksPerNode            *int32            `json:"tasks_per_node,omitempty"`
	TasksPerSocket          *int32            `json:"tasks_per_socket,omitempty"`
	TasksPerCore            *int32            `json:"tasks_per_core,omitempty"`
	TasksPerBoard           *int32            `json:"tasks_per_board,omitempty"`
	NTasksPerTres           *int32            `json:"ntasks_per_tres,omitempty"`
	MinimumCPUsPerNode      *int32            `json:"minimum_cpus_per_node,omitempty"`
	MemoryPerCPU            *uint64           `json:"memory_per_cpu,omitempty"`
	MemoryPerNode           *uint64           `json:"memory_per_node,omitempty"`
	TemporaryDiskPerNode    *int32            `json:"temporary_disk_per_node,omitempty"`
	SELinuxContext          *string           `json:"selinux_context,omitempty"`
	RequiredSwitches        *uint32           `json:"required_switches,omitempty"`
	StandardError           *string           `json:"standard_error,omitempty"`
	StandardInput           *string           `json:"standard_input,omitempty"`
	StandardOutput          *string           `json:"standard_output,omitempty"`
	WaitForSwitch           *int32            `json:"wait_for_switch,omitempty"`
	Wckey                   *string           `json:"wckey,omitempty"`
	X11                     *[]string         `json:"x11,omitempty"`
	X11MagicCookie          *string           `json:"x11_magic_cookie,omitempty"`
	X11TargetHost           *string           `json:"x11_target_host,omitempty"`
	X11TargetPort           *int32            `json:"x11_target_port,omitempty"`
}

type CronEntry struct {
	// TODO Define the fields for the v0.0.39_cron_entry struct if needed
}
