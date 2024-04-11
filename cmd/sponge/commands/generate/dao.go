package generate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zhufuyi/sponge/pkg/replacer"
	"github.com/zhufuyi/sponge/pkg/sql2code"
	"github.com/zhufuyi/sponge/pkg/sql2code/parser"
)

// DaoCommand generate dao code
func DaoCommand(parentName string) *cobra.Command {
	var (
		moduleName      string // go.mod module name
		outPath         string // output directory
		dbTables        string // table names
		isIncludeInitDB bool

		sqlArgs = sql2code.Args{
			Package:  "model",
			JSONTag:  true,
			GormType: true,
		}

		serverName     string // server name
		suitedMonoRepo bool   // whether the generated code is suitable for mono-repo
	)

	cmd := &cobra.Command{
		Use:   "dao",
		Short: "Generate dao code based on sql",
		Long: fmt.Sprintf(`generate dao code based on sql.

Examples:
  # generate dao code.
  sponge %s dao --module-name=yourModuleName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # generate dao code with multiple table names.
  sponge %s dao --module-name=yourModuleName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=t1,t2

  # generate dao code and specify the server directory, Note: code generation will be canceled when the latest generated file already exists.
  sponge %s dao --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --out=./yourServerDir

  # if you want the generated code to suited to mono-repo, you need to specify the parameter --suited-mono-repo=true --serverName=yourServerName
`, parentName, parentName, parentName),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			mdName, srvName, smr := getNamesFromOutDir(outPath)
			if mdName != "" {
				moduleName = mdName
				serverName = srvName
				suitedMonoRepo = smr
			} else if moduleName == "" {
				return fmt.Errorf(`required flag(s) "module-name" not set, use "sponge %s dao -h" for help`, parentName)
			}
			if suitedMonoRepo {
				if serverName == "" {
					return fmt.Errorf(`required flag(s) "server-name" not set, use "sponge %s dao -h" for help`, parentName)
				}
				serverName = convertServerName(serverName)
				outPath = changeOutPath(outPath, serverName)
			}

			tableNames := strings.Split(dbTables, ",")
			for count, tableName := range tableNames {
				if tableName == "" {
					continue
				}

				if sqlArgs.DBDriver == DBDriverMongodb {
					sqlArgs.IsEmbed = false
				}
				sqlArgs.DBTable = tableName
				codes, err := sql2code.Generate(&sqlArgs)
				if err != nil {
					return err
				}

				// control to generate the initialization db code only once
				if count == 0 && isIncludeInitDB {
					isIncludeInitDB = true
				} else {
					isIncludeInitDB = false
				}

				g := &daoGenerator{
					moduleName:      moduleName,
					dbDriver:        sqlArgs.DBDriver,
					isIncludeInitDB: isIncludeInitDB,
					codes:           codes,
					outPath:         outPath,

					serverName:     serverName,
					suitedMonoRepo: suitedMonoRepo,
				}
				outPath, err = g.generateCode()
				if err != nil {
					return err
				}
			}

			fmt.Printf(`
using help:
  move the folder "internal" to your project code folder.

`)
			fmt.Printf("generate \"dao\" code successfully, out = %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module-name", "m", "", "module-name is the name of the module in the go.mod file")
	//_ = cmd.MarkFlagRequired("module-name")
	cmd.Flags().StringVarP(&sqlArgs.DBDriver, "db-driver", "k", "mysql", "database driver, support mysql, mongodb, postgresql, tidb, sqlite")
	cmd.Flags().StringVarP(&sqlArgs.DBDsn, "db-dsn", "d", "", "database content address, e.g. user:password@(host:port)/database. Note: if db-driver=sqlite, db-dsn must be a local sqlite db file, e.g. --db-dsn=/tmp/sponge_sqlite.db") //nolint
	_ = cmd.MarkFlagRequired("db-dsn")
	cmd.Flags().StringVarP(&dbTables, "db-table", "t", "", "table name, multiple names separated by commas")
	_ = cmd.MarkFlagRequired("db-table")
	cmd.Flags().BoolVarP(&sqlArgs.IsEmbed, "embed", "e", false, "whether to embed gorm.model struct")
	cmd.Flags().StringVarP(&serverName, "server-name", "s", "", "server name")
	cmd.Flags().BoolVarP(&suitedMonoRepo, "suited-mono-repo", "l", false, "whether the generated code is suitable for mono-repo")
	cmd.Flags().IntVarP(&sqlArgs.JSONNamedType, "json-name-type", "j", 1, "json tags name type, 0:snake case, 1:camel case")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output directory, default is ./dao_<time>, "+
		"if you specify the directory where the web or microservice generated by sponge, the module-name flag can be ignored")
	cmd.Flags().BoolVarP(&isIncludeInitDB, "include-init-db", "i", false, "if true, includes mysql and redis initialization code")

	return cmd
}

type daoGenerator struct {
	moduleName      string
	dbDriver        string
	isIncludeInitDB bool
	codes           map[string]string
	outPath         string

	serverName     string
	suitedMonoRepo bool
}

func (g *daoGenerator) generateCode() (string, error) {
	subTplName := "dao"
	r := Replacers[TplNameSponge]
	if r == nil {
		return "", errors.New("r is nil")
	}

	// setting up template information
	subDirs := []string{ // only the specified subdirectory is processed, if empty or no subdirectory is specified, it means all files
		"internal/model", "internal/cache", "internal/dao",
	}
	ignoreDirs := []string{} // specify the directory in the subdirectory where processing is ignored
	var ignoreFiles []string
	switch strings.ToLower(g.dbDriver) {
	case DBDriverMysql, DBDriverPostgresql, DBDriverTidb, DBDriverSqlite:
		ignoreFiles = []string{ // specify the files in the subdirectory to be ignored for processing
			"init.go", "init_test.go", "init.go.mgo", // internal/model
			"doc.go", "cacheNameExample.go", "cacheNameExample_test.go", "cache/userExample.go.mgo", // internal/cache
			"dao/userExample.go.mgo", // internal/dao
		}
		if g.isIncludeInitDB {
			ignoreFiles = removeElement(ignoreFiles, "init.go")
		}
	case DBDriverMongodb:
		ignoreFiles = []string{ // specify the files in the subdirectory to be ignored for processing
			"init.go", "init_test.go", "init.go.mgo", // internal/model
			"doc.go", "cacheNameExample.go", "cacheNameExample_test.go", "cache/userExample.go", "cache/userExample_test.go", // internal/cache
			"dao/userExample_test.go", "dao/userExample.go", // internal/dao
		}
		if g.isIncludeInitDB {
			ignoreFiles = removeElement(ignoreFiles, "init.go.mgo")
		}
	default:
		return "", errors.New("unsupported db driver: " + g.dbDriver)
	}

	r.SetSubDirsAndFiles(subDirs)
	r.SetIgnoreSubDirs(ignoreDirs...)
	r.SetIgnoreSubFiles(ignoreFiles...)
	_ = r.SetOutputDir(g.outPath, subTplName)
	fields := g.addFields(r)
	r.SetReplacementFields(fields)
	if err := r.SaveFiles(); err != nil {
		return "", err
	}

	return r.GetOutputDir(), nil
}

// set fields
func (g *daoGenerator) addFields(r replacer.Replacer) []replacer.Field {
	var fields []replacer.Field

	fields = append(fields, deleteFieldsMark(r, modelFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoMgoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoTestFile, startMark, endMark)...)
	fields = append(fields, []replacer.Field{
		{ // replace the contents of the model/userExample.go file
			Old: modelFileMark,
			New: g.codes[parser.CodeTypeModel],
		},
		{
			Old: daoFileMark,
			New: g.codes[parser.CodeTypeDAO],
		},
		{
			Old: selfPackageName + "/" + r.GetSourcePath(),
			New: g.moduleName,
		},
		{
			Old: "github.com/zhufuyi/sponge",
			New: g.moduleName,
		},
		{
			Old: g.moduleName + "/pkg",
			New: "github.com/zhufuyi/sponge/pkg",
		},
		{
			Old: "init.go.mgo",
			New: "init.go",
		},
		{
			Old: "userExample.go.mgo",
			New: "userExample.go",
		},
		{
			Old:             "UserExample",
			New:             g.codes[parser.TableName],
			IsCaseSensitive: true,
		},
	}...)

	if g.suitedMonoRepo {
		fs := SubServerCodeFields(r.GetOutputDir(), g.moduleName, g.serverName)
		fields = append(fields, fs...)
	}

	return fields
}
