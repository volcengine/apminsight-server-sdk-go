package tags

type MetricTagKeysRegister interface {
	GetServerTagKeys() []string
	GetClientTagKeys() []string

	MergeTagKeysRegister(from MetricTagKeysRegister)
}

type tagKeysRegister struct {
	serverTagKeys []string
	clientTagKeys []string
}

var (
	builtinServerTags = []string{"from_service_type", "from_service", "http.status_code"}
	builtinClientTags = []string{"db.slow_query", "http.status_code"}
)

var builtinKeysReg = tagKeysRegister{
	serverTagKeys: builtinServerTags,
	clientTagKeys: builtinClientTags,
}

func GetBuiltinTagKeysRegister() MetricTagKeysRegister {
	return &builtinKeysReg
}

func NewMetricTagKeysRegister(sk, ck []string) MetricTagKeysRegister {
	return &tagKeysRegister{
		serverTagKeys: sk,
		clientTagKeys: ck,
	}
}

// MergeTagKeysRegister is called during tracer initialization and should not be called afterwards. so lock is not needed
func (t *tagKeysRegister) MergeTagKeysRegister(from MetricTagKeysRegister) {
	if from == nil {
		return
	}
	// deduplicate
	sm := make(map[string]struct{})
	cm := make(map[string]struct{})
	for _, k := range t.GetServerTagKeys() {
		sm[k] = struct{}{}
	}
	for _, k := range t.GetClientTagKeys() {
		cm[k] = struct{}{}
	}
	for _, k := range from.GetServerTagKeys() {
		sm[k] = struct{}{}
	}
	for _, k := range from.GetClientTagKeys() {
		cm[k] = struct{}{}
	}

	sl := make([]string, 0, len(sm))
	cl := make([]string, 0, len(cm))

	for k := range sm {
		sl = append(sl, k)
	}
	for k := range cm {
		cl = append(cl, k)
	}
	t.serverTagKeys = sl
	t.clientTagKeys = cl
}

func (t *tagKeysRegister) GetServerTagKeys() []string {
	return t.serverTagKeys
}

func (t *tagKeysRegister) GetClientTagKeys() []string {
	return t.clientTagKeys
}
