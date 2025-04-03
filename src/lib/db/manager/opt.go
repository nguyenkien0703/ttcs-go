package lib_db_manager

import "gorm.io/gorm"

/*
Mục đích: Định nghĩa một tác vụ cần thực hiện
Thành phần:
F: Hàm cần thực thi, nhận các tham số tùy ý và trả về lỗi (nếu có)
Args: Danh sách các tham số sẽ được truyền vào hàm F
Cách sử dụng: Được sử dụng để định nghĩa các tác vụ cần thực hiện trước hoặc sau khi lưu dữ liệu
*/
type Task struct {
	F func(...interface{}) error

	Args []interface{}
}

/*
Mục đích: Định nghĩa một tác vụ lưu trữ tùy chỉnh
Thành phần:
F: Hàm lưu trữ tùy chỉnh, nhận đối tượng database và các tham số tùy ý
Args: Danh sách các tham số sẽ được truyền vào hàm F
Cách sử dụng: Cho phép tùy chỉnh cách lưu trữ dữ liệu thay vì sử dụng các phương thức mặc định của GORM
3. Cấu trúc Sa
*/
type SaveFunctionTask struct {
	F    func(*gorm.DB, ...interface{}) error
	Args []interface{}
}

/*
	Mục đích: Định nghĩa các tùy chọn khi lưu trữ một model

Thành phần:
Fields: Danh sách các trường cần cập nhật (nếu rỗng, tất cả các trường sẽ được cập nhật)
PrepareTask: Tác vụ cần thực hiện trước khi lưu
SavedTask: Tác vụ cần thực hiện sau khi lưu
ForceInsert: Nếu true, bắt buộc thêm mới thay vì cập nhật
ForceUpdate: Nếu true, bắt buộc cập nhật thay vì thêm mới
SaveFunction: Hàm lưu trữ tùy chỉnh
Comment tiếng Nhật: "Tùy chọn khi lưu trữ"
*/
type SaveOptions struct {
	Fields       []string
	PrepareTask  *Task
	SavedTask    *Task
	ForceInsert  bool
	ForceUpdate  bool
	SaveFunction *SaveFunctionTask
}

/*
	Mục đích: Lưu trữ thông tin về một model đã được đặt lệnh lưu hoặc xóa

Thành phần:
Model: Đối tượng model cần lưu hoặc xóa
Fields: Danh sách các trường cần cập nhật
fieldMap: Map để kiểm tra nhanh xem một trường có trong danh sách cập nhật không
ForceInsert: Bắt buộc thêm mới
ForceUpdate: Bắt buộc cập nhật
PrepareTasks: Danh sách các tác vụ cần thực hiện trước khi lưu
SavedTasks: Danh sách các tác vụ cần thực hiện sau khi lưu
Priority: Độ ưu tiên khi thực hiện (số càng lớn càng ưu tiên)
SaveFunctions: Danh sách các hàm lưu trữ tùy chỉnh
*/
type ReservedInfo struct {
	Model         interface{}
	Fields        []string
	fieldMap      map[string]bool
	ForceInsert   bool
	ForceUpdate   bool
	PrepareTasks  []*Task
	SavedTasks    []*Task
	Priority      int
	SaveFunctions []*SaveFunctionTask
}
