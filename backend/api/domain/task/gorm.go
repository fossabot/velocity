package task

// type GORMTask struct {
// 	ID              string
// 	Commit          commit.GORMCommit `gorm:"ForeignKey:CommitReference"`
// 	CommitReference string
// 	TaskConfig      []byte // JSON of task name, parameters, steps etc.
// }

// func GORMTaskFromTask(t Task) GORMTask {
// 	taskConfig, err := json.Marshal(t.VTask)
// 	if err != nil {
// 		log.Printf("could not marshal task from %v", t)
// 		log.Fatal(err)
// 	}
// 	return GORMTask{
// 		ID:              t.ID,
// 		Commit:          commit.GORMCommitFromCommit(t.Commit),
// 		CommitReference: t.Commit.ID,
// 		TaskConfig:      taskConfig,
// 	}
// }

// func TaskFromGORMTask(g GORMTask) Task {
// 	var taskConfig velocity.Task
// 	err := json.Unmarshal(g.TaskConfig, &taskConfig)
// 	if err != nil {
// 		log.Printf("could not unmarshal task from %v", g)
// 		log.Fatal(err)
// 	}
// 	return Task{
// 		ID:     g.ID,
// 		Commit: commit.CommitFromGORMCommit(g.Commit),
// 		VTask:  taskConfig,
// 	}
// }

// // Expose CRUD operations (implement interface?) Implement repository funcs, as they will be used when we have caching.
// type gormRepository struct {
// 	logger *log.Logger
// 	gorm   *gorm.DB
// }

// func newGORMRepository(db *gorm.DB) *gormRepository {
// 	db.AutoMigrate(GORMTask{})
// 	return &gormRepository{
// 		logger: log.New(os.Stdout, "[gorm:task]", log.Lshortfile),
// 		gorm:   db,
// 	}
// }

// func (r *gormRepository) Save(t Task) Task {
// 	tx := r.gorm.Begin()

// 	gormTask := GORMTaskFromTask(t)

// 	err := tx.Where(&GORMTask{
// 		ID: t.ID,
// 	}).First(&GORMTask{}).Error
// 	if err != nil {
// 		err = tx.Create(&gormTask).Error
// 	} else {
// 		tx.Save(&gormTask)
// 	}

// 	tx.Commit()
// 	return t
// }

// func (r *gormRepository) Delete(t Task) {
// 	tx := r.gorm.Begin()

// 	gormTask := GORMTaskFromTask(t)

// 	if err := tx.Delete(gormTask).Error; err != nil {
// 		tx.Rollback()
// 		log.Fatal(err)
// 	}

// 	tx.Commit()
// }

// func (r *gormRepository) GetByProjectAndCommitAndID(p project.Project, c commit.Commit, ID string) (Task, error) {
// 	gormTask := GORMTask{}

// 	if r.gorm.
// 		Preload("Commit").
// 		Preload("Commit.Project").
// 		Preload("Commit.Branches").
// 		Preload("Commit.Branches.Project").
// 		Where(&GORMTask{
// 			CommitReference: c.ID,
// 			ID:              ID,
// 		}).
// 		First(&gormTask).RecordNotFound() {
// 		log.Printf("Could not find Task %s", ID)
// 		return Task{}, fmt.Errorf("could not find Task %s", ID)
// 	}

// 	return TaskFromGORMTask(gormTask), nil
// }

// func (r *gormRepository) GetAllByProjectAndCommit(p project.Project, c commit.Commit, q Query) ([]Task, uint64) {
// 	gormTasks := []GORMTask{}
// 	var count uint64

// 	r.gorm.
// 		Preload("Commit").
// 		Preload("Commit.Project").
// 		Preload("Commit.Branches").
// 		Preload("Commit.Branches.Project").
// 		Where(&GORMTask{
// 			CommitReference: c.ID,
// 		}).
// 		Find(&gormTasks).
// 		Count(&count)

// 	tasks := []Task{}
// 	for _, gTask := range gormTasks {
// 		tasks = append(tasks, TaskFromGORMTask(gTask))
// 	}

// 	return tasks, count
// }
