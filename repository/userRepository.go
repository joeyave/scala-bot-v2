package repository

import (
	"context"
	"fmt"
	"github.com/joeyave/scala-bot-v2/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

type UserRepository struct {
	mongoClient *mongo.Client
}

func NewUserRepository(mongoClient *mongo.Client) *UserRepository {
	return &UserRepository{
		mongoClient: mongoClient,
	}
}

func (r *UserRepository) FindAll() ([]*entity.User, error) {
	collection := r.mongoClient.Database(os.Getenv("MONGODB_DATABASE_NAME")).Collection("users")
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}

	var users []*entity.User
	err = cursor.All(context.TODO(), &users)
	return users, err
}

func (r *UserRepository) FindOneByID(ID int64) (*entity.User, error) {
	users, err := r.find(bson.M{
		"_id": ID,
	})
	if err != nil {
		return nil, err
	}

	return users[0], nil
}

func (r *UserRepository) FindOneByName(name string) (*entity.User, error) {
	users, err := r.find(
		bson.M{
			"name": name,
		},
		bson.M{
			"$limit": 1,
		},
	)
	if err != nil {
		return nil, err
	}

	return users[0], err
}

func (r *UserRepository) FindManyByIDs(IDs []int64) ([]*entity.User, error) {
	return r.find(bson.M{
		"_id": bson.M{
			"$in": IDs,
		},
	})
}

func (r *UserRepository) FindManyByBandID(bandID primitive.ObjectID) ([]*entity.User, error) {
	return r.find(bson.M{
		"bandId": bandID,
	})
}

func (r *UserRepository) find(m bson.M, opts ...bson.M) ([]*entity.User, error) {
	collection := r.mongoClient.Database(os.Getenv("MONGODB_DATABASE_NAME")).Collection("users")

	pipeline := bson.A{
		bson.M{
			"$match": m,
		},
		bson.M{
			"$lookup": bson.M{
				"from": "bands",
				"let":  bson.M{"bandId": "$bandId"},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$bandId"}}},
					},
					bson.M{
						"$lookup": bson.M{
							"from": "roles",
							"let":  bson.M{"bandId": "$_id"},
							"pipeline": bson.A{
								bson.M{
									"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$bandId", "$$bandId"}}},
								},
								bson.M{
									"$sort": bson.M{
										"priority": 1,
									},
								},
							},
							"as": "roles",
						},
					},
				},
				"as": "band",
			},
		},
		bson.M{
			"$unwind": bson.M{
				"path":                       "$band",
				"preserveNullAndEmptyArrays": true,
			},
		},
	}

	for _, o := range opts {
		pipeline = append(pipeline, o)
	}

	cur, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}

	var users []*entity.User
	err = cur.All(context.TODO(), &users)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return users, nil
}

func (r *UserRepository) UpdateOne(user entity.User) (*entity.User, error) {
	collection := r.mongoClient.Database(os.Getenv("MONGODB_DATABASE_NAME")).Collection("users")

	filter := bson.M{"_id": user.ID}

	user.ID = 0
	user.Band = nil
	update := bson.M{
		"$set": user,
	}

	after := options.After
	upsert := true
	opts := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	result := collection.FindOneAndUpdate(context.TODO(), filter, update, &opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	var newUser *entity.User
	err := result.Decode(&newUser)
	if err != nil {
		return nil, err
	}

	return r.FindOneByID(newUser.ID)
}

func (r *UserRepository) FindManyExtraByBandIDAndRoleID(bandID primitive.ObjectID, roleID primitive.ObjectID) ([]*entity.UserExtra, error) {
	pipeline := bson.A{
		bson.M{
			"$match": bson.M{
				"bandId": bandID,
			},
		},
		bson.M{
			"$lookup": bson.M{
				"from": "bands",
				"let":  bson.M{"bandId": "$bandId"},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$bandId"}}},
					},
					bson.M{
						"$lookup": bson.M{
							"from": "roles",
							"let":  bson.M{"bandId": "$_id"},
							"pipeline": bson.A{
								bson.M{
									"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$bandId", "$$bandId"}}},
								},
								bson.M{
									"$sort": bson.M{
										"priority": 1,
									},
								},
							},
							"as": "roles",
						},
					},
				},
				"as": "band",
			},
		},
		bson.M{
			"$unwind": bson.M{
				"path":                       "$band",
				"preserveNullAndEmptyArrays": true,
			},
		},
		bson.M{
			"$lookup": bson.M{
				"from": "events",
				"let":  bson.M{"userId": "$_id"},
				"pipeline": bson.A{
					bson.M{
						"$lookup": bson.M{
							"from": "memberships",
							"let":  bson.M{"eventId": "$_id"},
							"pipeline": bson.A{
								bson.M{
									"$match": bson.M{"$and": bson.A{
										bson.M{"$expr": bson.M{"$eq": bson.A{"$eventId", "$$eventId"}}},
										bson.M{"$expr": bson.M{"$eq": bson.A{"$roleId", roleID}}},
									}},
								},
							},
							"as": "memberships",
						},
					},
					bson.M{
						"$match": bson.M{"$expr": bson.M{"$in": bson.A{"$$userId", "$memberships.userId"}}},
					},
					bson.M{
						"$sort": bson.M{
							"time": -1,
						},
					},
				},
				"as": "events",
			},
		},

		bson.M{
			"$addFields": bson.M{
				"lastEventTime": bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$eq": bson.A{bson.M{"$size": "$events"}, 0},
						},
						"then": time.Now().AddDate(10, 0, 0),
						"else": bson.M{
							"$first": "$events.time",
						},
					},
				},
			},
		},

		bson.M{
			"$sort": bson.M{
				"lastEventTime": -1,
			},
		},
	}

	return r.findWithExtra(pipeline)
}

func (r *UserRepository) FindManyExtraByBandID(bandID primitive.ObjectID) ([]*entity.UserExtra, error) {
	pipeline := bson.A{
		bson.M{
			"$match": bson.M{
				"bandId": bandID,
			},
		},
		bson.M{
			"$lookup": bson.M{
				"from": "bands",
				"let":  bson.M{"bandId": "$bandId"},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$bandId"}}},
					},
					bson.M{
						"$lookup": bson.M{
							"from": "roles",
							"let":  bson.M{"bandId": "$_id"},
							"pipeline": bson.A{
								bson.M{
									"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$bandId", "$$bandId"}}},
								},
								bson.M{
									"$sort": bson.M{
										"priority": 1,
									},
								},
							},
							"as": "roles",
						},
					},
				},
				"as": "band",
			},
		},
		bson.M{
			"$unwind": bson.M{
				"path":                       "$band",
				"preserveNullAndEmptyArrays": true,
			},
		},
		bson.M{
			"$lookup": bson.M{
				"from": "events",
				"let":  bson.M{"userId": "$_id"},
				"pipeline": bson.A{
					bson.M{
						"$lookup": bson.M{
							"from": "memberships",
							"let":  bson.M{"eventId": "$_id"},
							"pipeline": bson.A{
								bson.M{
									"$match": bson.M{
										"$expr": bson.M{"$eq": bson.A{"$eventId", "$$eventId"}},
									},
								},
								bson.M{
									"$lookup": bson.M{
										"from": "roles",
										"let":  bson.M{"roleId": "$roleId"},
										"pipeline": bson.A{
											bson.M{
												"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$roleId"}}},
											},
										},
										"as": "role",
									},
								},
								bson.M{
									"$unwind": bson.M{
										"path":                       "$role",
										"preserveNullAndEmptyArrays": true,
									},
								},
								bson.M{
									"$sort": bson.M{
										"role.priority": 1,
									},
								},
							},
							"as": "memberships",
						},
					},
					bson.M{
						"$match": bson.M{"$expr": bson.M{"$in": bson.A{"$$userId", "$memberships.userId"}}},
					},
				},
				"as": "events",
			},
		},

		bson.M{
			"$addFields": bson.M{
				"eventsSize": bson.M{"$size": "$events"},
			},
		},

		bson.M{
			"$sort": bson.M{
				"eventsSize": -1,
			},
		},
	}

	return r.findWithExtra(pipeline)
}

func (r *UserRepository) findWithExtra(pipeline bson.A) ([]*entity.UserExtra, error) {
	collection := r.mongoClient.Database(os.Getenv("MONGODB_DATABASE_NAME")).Collection("users")

	cur, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}

	var users []*entity.UserExtra
	err = cur.All(context.TODO(), &users)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return users, nil
}
