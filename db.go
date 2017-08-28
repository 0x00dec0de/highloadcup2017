package main

import (
	"bytes"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"time"

	"sync"

	_ "github.com/go-sql-driver/mysql"
)

var NotFound = fmt.Errorf("Not found")
var CannotParse = fmt.Errorf("Cannot parse")

type Database struct {
	users, locations, visits sync.Map
}

func (DB *Database) NewUser(user *User) error {
	Log.Infof("Inserting user with id %d", user.Id)
	DB.users.Store(user.Id, user)
	return nil
}

func (DB *Database) GetUser(id int) (*User, error) {
	Log.Infof("Getting user with id %d", id)

	user, ok := DB.users.Load(id)
	if !ok {
		return &User{}, NotFound
	}

	return user.(*User), nil
}

func (DB *Database) UpdateUser(user *User, id int) error {
	Log.Infof("Updating user with id %d", id)

	//DB.users.Store(user.Id, user)
	return nil
}

func (DB *Database) NewLocation(loc *Location) error {
	Log.Infof("Inserting location with id %d", loc.Id)
	DB.locations.Store(loc.Id, loc)
	return nil
}

func (DB *Database) GetLocation(id int) (*Location, error) {
	Log.Infof("Getting location with id %d", id)

	loc, ok := DB.locations.Load(id)
	if !ok {
		return &Location{}, NotFound
	}

	return loc.(*Location), nil
}

func (DB *Database) UpdateLocation(loc *Location, id int) error {
	Log.Infof("Updating location with id %d", id)

	//DB.locations.Store(loc.Id, loc)
	return nil
}

func (DB *Database) NewVisit(visit *Visit) error {
	Log.Infof("Inserting visit with id %d", visit.Id)

	DB.visits.Store(visit.Id, visit)
	usr, err := DB.GetUser(visit.User)
	if err != nil {
		return fmt.Errorf("Cannot get user %d. Reason %s", visit.User, err)
	}
	usr.Visits.Add(visit.Id)
	loc, err := DB.GetLocation(visit.Location)
	if err != nil {
		return fmt.Errorf("Cannot get location %d. Reason %s", visit.Location, err)
	}
	loc.Visits.Add(visit.Id)
	return nil
}

func (DB *Database) GetVisit(id int) (*Visit, error) {
	Log.Infof("Getting visit with id %d", id)

	v, ok := DB.visits.Load(id)
	if !ok {
		return &Visit{}, NotFound
	}
	return v.(*Visit), nil
}

func (DB *Database) UpdateVisit(visit *Visit, id int) error {
	Log.Infof("Updating visit with id %d", id)

	//DB.visits.Store(visit.Id, visit)
	return nil
}

//select visited_at, mark, place from (select * from visits where id = 1) as v inner join locations on locations.id = v.location where distance < 1000000;
func (DB *Database) GetVisitsFilter(id int, filters url.Values) ([]UserVisits, error) {
	result := make([]UserVisits, 0)
	var err error

	usr, err := DB.GetUser(id)
	if err == NotFound {
		return result, NotFound
	}

	fromDateStr := filters.Get("fromDate")
	fromDate, err := strconv.Atoi(fromDateStr)
	if err != nil {
		if fromDateStr != "" {
			return result, CannotParse
		}
		fromDate = math.MinInt32
	}

	toDateStr := filters.Get("toDate")
	toDate, err := strconv.Atoi(toDateStr)
	if err != nil {
		if toDateStr != "" {
			return result, CannotParse
		}
		toDate = math.MaxInt32
	}

	country := filters.Get("country")

	toDistanceStr := filters.Get("toDistance")
	toDistance, err := strconv.Atoi(toDistanceStr)
	if err != nil {
		if toDistanceStr != "" {
			return result, CannotParse
		}
		toDistance = math.MaxInt32
	}

	usr.Visits.ForEach(func(id int) bool {
		visit, err := DB.GetVisit(id)
		if err == nil && visit.VisitedAt > fromDate && visit.VisitedAt < toDate {
			location, err := DB.GetLocation(visit.Location)
			if err == nil && (country == "" || location.Country == country) && location.Distance < toDistance {
				result = append(result, UserVisits{
					VisitedAt: visit.VisitedAt,
					Mark:      visit.Mark,
					Place:     location.Place,
				})
			}
		}
		return true
	})

	/*DB.visits.Range(func(key, v interface{}) bool {
		visit := v.(Visit)
		if visit.User == id {
			if visit.VisitedAt > fromDate && visit.VisitedAt < toDate {
				location, err := DB.GetLocation(visit.Location)
				if err == nil && (country == "" || location.Country == country) && location.Distance < toDistance {
					result = append(result, UserVisits{
						VisitedAt: visit.VisitedAt,
						Mark:      visit.Mark,
						Place:     location.Place,
					})
				}
			}
		}
		return true
	})*/

	sorter := UserVisitsSorter{
		Data: result,
	}
	sort.Sort(sorter)

	return result, nil
}

//select AVG(mark) from
//(select user, visitedAt, mark from (select * from visits where location=2) as v inner join locations on locations.id = v.location where visitedAt>500) as t inner join users on users.id = t.user where gender = "f";
//
func (DB *Database) GetAverage(id int, filters url.Values) (float32, error) {
	var marks float32
	var count int

	loc, err := DB.GetLocation(id)
	if err != nil {
		return 0.0, NotFound
	}

	fromDateStr := filters.Get("fromDate")
	fromDate, err := strconv.Atoi(fromDateStr)
	if err != nil {
		if fromDateStr != "" {
			return 0.0, CannotParse
		}
		fromDate = math.MinInt32
	}

	toDateStr := filters.Get("toDate")
	toDate, err := strconv.Atoi(toDateStr)
	if err != nil {
		if toDateStr != "" {
			return 0.0, CannotParse
		}
		toDate = math.MaxInt32
	}

	fromAgeStr := filters.Get("fromAge")
	fromAge, err := strconv.Atoi(fromAgeStr)
	if err != nil {
		if fromAgeStr != "" {
			return 0.0, CannotParse
		}
		fromAge = 0
	}

	toAgeStr := filters.Get("toAge")
	toAge, err := strconv.Atoi(toAgeStr)
	if err != nil {
		if toAgeStr != "" {
			return 0.0, CannotParse
		}
		toAge = -1
	}

	gender := filters.Get("gender")
	if gender != "m" && gender != "f" && gender != "" {
		return 0.0, CannotParse
	}

	loc.Visits.ForEach(func(id int) bool {
		visit, err := DB.GetVisit(id)
		if err == nil && visit.VisitedAt > fromDate && visit.VisitedAt < toDate {
			user, err := DB.GetUser(visit.User)
			if err == nil {
				Log.Warnf("Found user for that visit %#v", user)
				if time.Unix(int64(user.Birthdate), 0).AddDate(fromAge, 0, 0).Before(ts) {
					Log.Warnf("Before ok %v %v", time.Unix(int64(user.Birthdate), 0).AddDate(fromAge, 0, 0), ts)
					if toAge == -1 || time.Unix(int64(user.Birthdate), 0).AddDate(toAge, 0, 0).After(ts) {
						Log.Warnf("Ater ok")
						if gender == "" || user.Gender == gender {
							Log.Infof("Adding %f", float32(visit.Mark))
							marks += float32(visit.Mark)
							count += 1
							Log.Infof("Marks %f %d", marks, count)
						}
					}
				}
			}
		}
		return true
	})

	/*DB.visits.Range(func(key, v interface{}) bool {
		visit := v.(Visit)
		if visit.Location == id {
			if visit.VisitedAt > fromDate && visit.VisitedAt < toDate {
				user, err := DB.GetUser(visit.User)
				if err == nil {
					Log.Warnf("Found user for that visit %#v", user)
					if time.Unix(int64(user.Birthdate), 0).AddDate(fromAge, 0, 0).Before(ts) {
						Log.Warnf("Before ok %v %v", time.Unix(int64(user.Birthdate), 0).AddDate(fromAge, 0, 0), ts)
						if toAge == -1 || time.Unix(int64(user.Birthdate), 0).AddDate(toAge, 0, 0).After(ts) {
							Log.Warnf("Ater ok")
							if gender == "" || user.Gender == gender {
								Log.Infof("Adding %f", float32(visit.Mark))
								marks += float32(visit.Mark)
								count += 1
								Log.Infof("Marks %f %d", marks, count)
							}
						}
					}
				}
			}
		}
		return true
	})*/

	if count == 0 {
		return 0.0, nil
	}

	return marks / float32(count), nil
}

func generateWhereClasure(filters url.Values) (string, error) {
	var buf bytes.Buffer
	if filters.Get("fromDate") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		if _, err := strconv.Atoi(filters.Get("fromDate")); err != nil {
			return "", fmt.Errorf("Cannot convert fromDate %s", filters.Get("fromDate"))
		}
		buf.WriteString("visitedAt > " + filters.Get("fromDate"))
	}
	if filters.Get("toDate") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		if _, err := strconv.Atoi(filters.Get("toDate")); err != nil {
			return "", fmt.Errorf("Cannot convert toDate %s", filters.Get("toDate"))
		}
		buf.WriteString("visitedAt < " + filters.Get("toDate"))
	}
	if filters.Get("country") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		buf.WriteString("country = \"" + filters.Get("country") + "\"")
	}
	if filters.Get("toDistance") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		if _, err := strconv.Atoi(filters.Get("toDistance")); err != nil {
			return "", fmt.Errorf("Cannot convert toDistance %s", filters.Get("toDistance"))
		}
		buf.WriteString("distance < " + filters.Get("toDistance"))
	}

	if buf.Len() > 0 {
		return "WHERE " + buf.String(), nil
	}
	return "", nil
}

func generateWhereClasureAvgInner(filters url.Values) (string, error) {
	var buf bytes.Buffer
	if filters.Get("fromDate") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		if _, err := strconv.Atoi(filters.Get("fromDate")); err != nil {
			return "", fmt.Errorf("Cannot convert fromDate %s", filters.Get("fromDate"))
		}
		buf.WriteString("visitedAt > " + filters.Get("fromDate"))
	}
	if filters.Get("toDate") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		if _, err := strconv.Atoi(filters.Get("toDate")); err != nil {
			return "", fmt.Errorf("Cannot convert toDate %s", filters.Get("toDate"))
		}
		buf.WriteString("visitedAt < " + filters.Get("toDate"))
	}

	if buf.Len() > 0 {
		return "WHERE " + buf.String(), nil
	}
	return "", nil
}

func generateWhereClasureAvgOutter(filters url.Values) (string, error) {
	var buf bytes.Buffer
	if filters.Get("fromAge") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		fromAge, err := strconv.Atoi(filters.Get("fromAge"))
		if err != nil {
			return "", fmt.Errorf("Cannot parse fromAge. %s Reason %s", filters.Get("fromAge"), err)
		}
		fromDateAge := time.Unix(0, 0).AddDate(fromAge, 0, 0).Unix()
		buf.WriteString("birthdate + " + strconv.FormatInt(fromDateAge, 10) + " < " + strconv.FormatInt(time.Now().Unix(), 10))
	}
	if filters.Get("toAge") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		toAge, err := strconv.Atoi(filters.Get("toAge"))
		if err != nil {
			return "", fmt.Errorf("Cannot parse toAge. %s Reason %s", filters.Get("toAge"), err)
		}
		toDateAge := time.Unix(0, 0).AddDate(toAge, 0, 0).Unix()
		buf.WriteString("birthdate + " + strconv.FormatInt(toDateAge, 10) + " > " + strconv.FormatInt(time.Now().Unix(), 10))
	}
	if filters.Get("gender") != "" {
		if buf.Len() != 0 {
			buf.WriteString(" and ")
		}
		buf.WriteString("gender = \"" + filters.Get("gender") + "\"")
	}

	if buf.Len() > 0 {
		return "WHERE " + buf.String(), nil
	}
	return "", nil
}

func DatabaseInit() (*Database, error) {
	db := Database{}
	return &db, nil
}
