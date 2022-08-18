package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/leonardodelira/gojwt/database"
	"github.com/leonardodelira/gojwt/helpers"
	"github.com/leonardodelira/gojwt/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var validate = validator.New()

func HashPassword(userPassword string) string { 
	hash, err := bcrypt.GenerateFromPassword([]byte(userPassword), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(hash)
}

func VerifyPassword(hashPassword string, providePassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(providePassword))
	if err != nil {
		return false, "Incorrect password"
	}
	return true, ""
}

func SignUp() gin.HandlerFunc {
	return func (c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email, "phone": user.Phone})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error":"error occured while try sign up"})
		}

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error":"email or phone number already exists"})
		}

		password := HashPassword(*user.Password)
		user.Password = &password
		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID() 
		user.User_id = user.ID.Hex()
		token, refreshToken, _ := helpers.GenerateAllTokens(user)
		user.Token = &token
		user.Refresh_token = &refreshToken
		
		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			msg := fmt.Sprintf("User item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}

func Login() gin.HandlerFunc {
	return func (c *gin.Context)  {
		var user models.User
		
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var userFound models.User

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&userFound)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if userFound.Email == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
			return
		}

		passwordIsValid, msg := VerifyPassword(*userFound.Password, *user.Password)
		if passwordIsValid != true {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}

		token, refreshToken, _ := helpers.GenerateAllTokens(userFound)
		helpers.UpdateAllTokens(token, refreshToken, userFound.User_id)
		
		err = userCollection.FindOne(ctx, bson.M{"user_id": userFound.User_id}).Decode(&userFound)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		*userFound.Password = "";
		c.JSON(http.StatusOK, &userFound)
	}
}

func GetUsers() gin.HandlerFunc{
	return func(c *gin.Context){
		if err := helpers.CheckUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error":err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage <1{
			recordPerPage = 10
		}
		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 !=nil || page<1{
			page = 1
		}

		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))

		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}
		groupStage := bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "_id", Value: "null"}}}, 
			{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}}, 
			{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
		}}}

		projectStage := bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "total_count", Value: 1},
				{Key: "user_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}},
			}}}

		result,err := userCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage, projectStage})
		defer cancel()
		
		if err!=nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error":"error occured while listing user items"})
		}
		
		var allusers []bson.M
		if err = result.All(ctx, &allusers); err!=nil{
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allusers[0])}
}

func GetUser() gin.HandlerFunc {
	return func (c *gin.Context)  {
		userId := c.Param("userId")

		if err := helpers.MatchUserTypeToUuid(c, userId); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}