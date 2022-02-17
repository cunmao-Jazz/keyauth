package impl

import (
	"context"
	"fmt"

	"github.com/cunmao-Jazz/keyauth/apps/book"

	"github.com/infraboard/mcube/exception"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *service) save(ctx context.Context, ins *book.Book) error {
	// InsertOne, 插入单个文档
	// InserMany 插入多个文档

	if _, err := s.col.InsertOne(ctx, ins); err != nil {
		return exception.NewInternalServerError("inserted book(%s) document error, %s",
			ins.Data.Name, err)
	}
	return nil
}

func (s *service) get(ctx context.Context, id string) (*book.Book, error) {
	//{"name":"wen"}
	filter := bson.M{"_id": id}

	ins := book.NewDefaultBook()
	// FindOne, 从集合里面获取一个文档
	//filter，mongo 采用 JSON格式的过滤条件的参数
	//Decode, 根据Sruct tag(bson),把文档里面的值 赋值给对象
	if err := s.col.FindOne(ctx, filter).Decode(ins); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, exception.NewNotFound("book %s not found", id)
		}

		return nil, exception.NewInternalServerError("find book %s error, %s", id, err)
	}

	return ins, nil
}

func newQueryBookRequest(r *book.QueryBookRequest) *queryBookRequest {
	return &queryBookRequest{
		r,
	}
}

type queryBookRequest struct {
	*book.QueryBookRequest
}

//倒叙查询;desc
//分页 offset limit ----> skip,limit
func (r *queryBookRequest) FindOptions() *options.FindOptions {
	pageSize := int64(r.Page.PageSize)
	skip := int64(r.Page.PageSize) * int64(r.Page.PageNumber-1)

	opt := &options.FindOptions{
		// {"create_at": -1}
		// -1 倒叙
		// 1 正序
		Sort: bson.D{
			{Key: "create_at", Value: -1},
		},
		Limit: &pageSize,
		Skip:  &skip,
	}

	return opt
}


// 关键字搜索
func (r *queryBookRequest) FindFilter() bson.M {
	filter := bson.M{}
	if r.Keywords != "" {
		// bson.M{}, 相当于 AND，{"name":"zs","gender":1 "$or":[]}
		//bson.A 表示一个Array对象
		filter["$or"] = bson.A{
			//如何做正则匹配: $regex {"filed1":{"regex":"z*"}}
			// $options, 正则匹配的参数，i 忽略大小写,m代表多个
			bson.M{"data.name": bson.M{"$regex": r.Keywords, "$options": "im"}},
			bson.M{"data.author": bson.M{"$regex": r.Keywords, "$options": "im"}},
		}
	}
	return filter
}

func (s *service) query(ctx context.Context, req *queryBookRequest) (*book.BookSet, error) {
	// Find 查询多个文档
	resp, err := s.col.Find(ctx, req.FindFilter(), req.FindOptions())

	if err != nil {
		return nil, exception.NewInternalServerError("find book error, error is %s", err)
	}

	set := book.NewBookSet()
	// 循环
	for resp.Next(ctx) {
		ins := book.NewDefaultBook()
		if err := resp.Decode(ins); err != nil {
			return nil, exception.NewInternalServerError("decode book error, error is %s", err)
		}

		set.Add(ins)
	}

	// count
	// 如何统计数量
	count, err := s.col.CountDocuments(ctx, req.FindFilter())
	if err != nil {
		return nil, exception.NewInternalServerError("get book count error, error is %s", err)
	}
	set.Total = count

	return set, nil
}

func (s *service) update(ctx context.Context, ins *book.Book) error {
	// UpdateByID 根据 id update
	//updateMary UPDATE WHERE ...
	if _, err := s.col.UpdateByID(ctx, ins.Id, ins); err != nil {
		return exception.NewInternalServerError("inserted book(%s) document error, %s",
			ins.Data.Name, err)
	}

	return nil
}

func (s *service) deleteBook(ctx context.Context, ins *book.Book) error {
	if ins == nil || ins.Id == "" {
		return fmt.Errorf("book is nil")
	}
	//DeleteOne filter 即使找到多个，只删除第一个
	//DeleteMany filter 匹配到的都删除
	result, err := s.col.DeleteOne(ctx, bson.M{"_id": ins.Id})
	if err != nil {
		return exception.NewInternalServerError("delete book(%s) error, %s", ins.Id, err)
	}

	//影响到的文档数量
	if result.DeletedCount == 0 {
		return exception.NewNotFound("book %s not found", ins.Id)
	}

	return nil
}
