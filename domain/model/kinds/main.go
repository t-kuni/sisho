package kinds

type KindName string

const KindNameExamples KindName = "examples"
const KindNameImplementations KindName = "implementations"
const KindNameSpecifications KindName = "specifications"
const KindNameDependencies KindName = "dependencies"
const KindNameKnowledgeList KindName = "knowledge-list"

type Kind struct {
	Name        KindName
	Description string
}

var kinds map[KindName]Kind

func init() {
	kinds = map[KindName]Kind{
		KindNameExamples: {
			Name:        KindNameExamples,
			Description: "コード例です。これを参考にして実装を進めてください。",
		},
		KindNameImplementations: {
			Name:        KindNameImplementations,
			Description: "利用可能な実装です。必要に応じて利用してください。",
		},
		KindNameSpecifications: {
			Name:        KindNameSpecifications,
			Description: "この仕様を満たすように実装してください。",
		},
		KindNameDependencies: {
			Name:        KindNameDependencies,
			Description: "利用可能なライブラリの一覧です。必要に応じて利用してください。",
		},
		KindNameKnowledgeList: {
			Name:        KindNameKnowledgeList,
			Description: "",
		},
	}
}

func GetKind(name KindName) (Kind, bool) {
	kind, ok := kinds[name]
	return kind, ok
}
