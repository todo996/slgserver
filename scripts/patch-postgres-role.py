from pathlib import Path

path = Path("server/slgserver/controller/role.go")
source = path.read_text(encoding="utf-8")

replacements = [
    (
        '\t"math/rand"\n\t"time"\n\n\t"github.com/go-sql-driver/mysql"',
        '\t"errors"\n\t"math/rand"\n\t"strings"\n\t"time"\n\t"unicode"\n\t"unicode/utf8"\n\n\t"github.com/go-sql-driver/mysql"',
    ),
    (
        '\t"github.com/goinggo/mapstructure"\n',
        '\t"github.com/goinggo/mapstructure"\n\t"github.com/lib/pq"\n',
    ),
    (
        '\trspObj.Role.UId = reqObj.UId\n\n\tr := make([]model.Role, 0)',
        '\trspObj.Role.UId = reqObj.UId\n\n'
        '\treqObj.NickName = strings.TrimSpace(reqObj.NickName)\n'
        '\tif !validRoleName(reqObj.NickName) {\n'
        '\t\trsp.Body.Code = constant.InvalidParam\n'
        '\t\treturn\n'
        '\t}\n\n'
        '\tr := make([]model.Role, 0)',
    ),
    (
        '\t\tif _, err := db.MasterDB.Insert(role); err != nil {\n'
        '\t\t\tlog.DefaultLog.Info("role  create error",\n'
        '\t\t\t\tzap.Int("uid", reqObj.UId), zap.Error(err))\n'
        '\t\t\te, _ := err.(*mysql.MySQLError)\n'
        '\t\t\tif 1062 == e.Number {\n'
        '\t\t\t\trsp.Body.Code = constant.RoleNameExist\n'
        '\t\t\t} else {\n'
        '\t\t\t\trsp.Body.Code = constant.DBError\n'
        '\t\t\t}\n'
        '\t\t} else {',
        '\t\tif _, err := db.MasterDB.Insert(role); err != nil {\n'
        '\t\t\tlog.DefaultLog.Info("role create error",\n'
        '\t\t\t\tzap.Int("uid", reqObj.UId), zap.Error(err))\n'
        '\t\t\tif isUniqueViolation(err) {\n'
        '\t\t\t\trsp.Body.Code = constant.RoleNameExist\n'
        '\t\t\t} else {\n'
        '\t\t\t\trsp.Body.Code = constant.DBError\n'
        '\t\t\t}\n'
        '\t\t} else {',
    ),
    (
        'func (this *Role) roleList(req *net.WsMsgReq, rsp *net.WsMsgRsp) {',
        '''func validRoleName(name string) bool {
\truneCount := utf8.RuneCountInString(name)
\tif runeCount < 2 || runeCount > 20 {
\t\treturn false
\t}

\tfor _, char := range name {
\t\tif unicode.IsLetter(char) || unicode.IsNumber(char) || unicode.IsSpace(char) || char == '_' || char == '-' {
\t\t\tcontinue
\t\t}
\t\treturn false
\t}
\treturn true
}

func isUniqueViolation(err error) bool {
\tvar postgresError *pq.Error
\tif errors.As(err, &postgresError) && postgresError.Code == "23505" {
\t\treturn true
\t}

\tvar mysqlError *mysql.MySQLError
\treturn errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func (this *Role) roleList(req *net.WsMsgReq, rsp *net.WsMsgRsp) {''',
    ),
]

for old, new in replacements:
    if source.count(old) != 1:
        raise SystemExit(f"Không tìm thấy đúng một đoạn cần vá: {old[:80]!r}")
    source = source.replace(old, new, 1)

path.write_text(source, encoding="utf-8")
print("Đã vá tạo nhân vật cho PostgreSQL và tên Unicode.")
