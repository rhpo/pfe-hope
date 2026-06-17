package tests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"pfe-backend/internal/config"
	"pfe-backend/internal/entity"
	"pfe-backend/internal/handler"
	"pfe-backend/internal/repository"
	"pfe-backend/internal/service"
	"pfe-backend/internal/shared/middleware"
	"pfe-backend/internal/shared/notify"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

type TestHelper struct {
	App   *fiber.App
	DB    *sql.DB
	Cfg   *config.Config
	Admin string
}

const (
	SeedAdminID        int64 = 1
	SeedTeacherISIL1ID int64 = 2
	SeedTeacherISIL2ID int64 = 3
	SeedTeacherCHIM1ID int64 = 4
	SeedStudentISIL1ID int64 = 5
	SeedStudentISIL2ID int64 = 6
	SeedStudentCHIM1ID int64 = 7
	SeedStudentISIL4ID int64 = 8
	SeedCompany1ID     int64 = 9
)

func NewTestHelper() *TestHelper {
	os.Setenv("ENV", "development")
	os.Setenv("SUPABASE_URL", "https://test.supabase.co")
	os.Setenv("SUPABASE_SERVICE_ROLE_KEY", "test-key")
	os.Setenv("RESEND_API_KEY", "test-resend-key")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("DATABASE_PATH", ":memory:")
	os.Setenv("PORT", "9099")

	cfg := config.Load()

	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		panic(fmt.Sprintf("Erreur ouverture DB test: %v", err))
	}

	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("Erreur ping DB test: %v", err))
	}

	if err := runTestMigrations(db); err != nil {
		panic(fmt.Sprintf("Erreur migration test: %v", err))
	}

	if err := runTestSeed(db); err != nil {
		panic(fmt.Sprintf("Erreur seed test: %v", err))
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(map[string]any{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
	}))

	profileRepo := repository.NewProfileRepository(db)
	teacherRepo := repository.NewTeacherRepository(db)
	studentRepo := repository.NewStudentRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	domainRepo := repository.NewDomainRepository(db)
	specialityRepo := repository.NewSpecialityRepository(db)
	promotionRepo := repository.NewPromotionRepository(db)
	academicYearRepo := repository.NewAcademicYearRepository(db)
	pfeSubjectRepo := repository.NewPfeSubjectRepository(db)
	wishRepo := repository.NewWishRepository(db)
	pfeAssignmentRepo := repository.NewPfeAssignmentRepository(db)
	progressRepo := repository.NewProgressReportRepository(db)
	defenseJuryRepo := repository.NewDefenseJuryRepository(db)
	defenseRepo := repository.NewDefenseRepository(db)
	juryGradeRepo := repository.NewJuryGradeRepository(db)
	supEvalRepo := repository.NewSupervisorEvaluationRepository(db)
	companyReportRepo := repository.NewCompanyReportRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	departmentRepo := repository.NewDepartmentRepository(db)

	authService := service.NewAuthService(profileRepo, teacherRepo, studentRepo, companyRepo, cfg)
	testNotifier := notify.New(notificationRepo, profileRepo, "test-resend-key")
	adminService := service.NewAdminService(
		profileRepo, teacherRepo, studentRepo, companyRepo, departmentRepo,
		domainRepo, specialityRepo, promotionRepo, academicYearRepo,
		pfeSubjectRepo, wishRepo, pfeAssignmentRepo,
		progressRepo, defenseJuryRepo, defenseRepo,
		juryGradeRepo, supEvalRepo, companyReportRepo,
		notificationRepo, auditLogRepo, testNotifier, "./uploads",
	)

	teacherService := service.NewTeacherService(
		profileRepo, teacherRepo, studentRepo, companyRepo, specialityRepo,
		pfeSubjectRepo, wishRepo, pfeAssignmentRepo, progressRepo,
		defenseJuryRepo, defenseRepo, juryGradeRepo, supEvalRepo, notificationRepo, academicYearRepo, testNotifier,
	)

	studentService := service.NewStudentService(
		profileRepo, studentRepo, teacherRepo, companyRepo, specialityRepo,
		pfeSubjectRepo, wishRepo, pfeAssignmentRepo, progressRepo,
		defenseRepo, defenseJuryRepo, supEvalRepo, notificationRepo, academicYearRepo,
	)
	companyService := service.NewCompanyService(
		profileRepo, companyRepo, teacherRepo, studentRepo, specialityRepo,
		pfeSubjectRepo, wishRepo, pfeAssignmentRepo, progressRepo,
		supEvalRepo, companyReportRepo, notificationRepo, academicYearRepo, testNotifier,
	)

	notifier := notify.New(notificationRepo, profileRepo, "test-resend-key")

	authHandler := handler.NewAuthHandler(authService, cfg, notifier)
	adminHandler := handler.NewAdminHandler(adminService, notifier)
	teacherHandler := handler.NewTeacherHandler(teacherService, notifier)
	studentHandler := handler.NewStudentHandler(studentService, notifier)
	companyHandler := handler.NewCompanyHandler(companyService, notifier)
	uploadHandler := handler.NewUploadHandler(profileRepo, companyRepo, "./uploads")

	app.Get("/uploads/*", static.New("./uploads"))

	api := app.Group("/api")

	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(map[string]any{"success": true, "data": map[string]any{"status": "ok"}})
	})

	ref := api.Group("/ref", middleware.AuthRequired(cfg))
	ref.Get("/domains", func(c fiber.Ctx) error {
		domains, err := domainRepo.FindAll()
		if err != nil {
			return c.Status(500).JSON(map[string]any{"success": false, "error": "Erreur serveur"})
		}
		if domains == nil {
			domains = []*entity.Domain{}
		}
		return c.JSON(map[string]any{"success": true, "data": domains})
	})
	ref.Get("/specialities", func(c fiber.Ctx) error {
		specialities, err := specialityRepo.FindAll()
		if err != nil {
			return c.Status(500).JSON(map[string]any{"success": false, "error": "Erreur serveur"})
		}
		if specialities == nil {
			specialities = []*entity.Speciality{}
		}
		return c.JSON(map[string]any{"success": true, "data": specialities})
	})

	api.Post("/auth/dev-login", authHandler.DevLogin)
	auth := api.Group("/auth")
	auth.Use(middleware.AuthRequired(cfg))
	auth.Get("/me", authHandler.Me)
	auth.Post("/logout", authHandler.Logout)

	api.Post("/profile/avatar", middleware.AuthRequired(cfg), uploadHandler.UploadAvatar)

	upload := api.Group("/upload", middleware.AuthRequired(cfg))
	upload.Post("/company-logo", uploadHandler.UploadCompanyLogo)
	upload.Post("/memoire", uploadHandler.UploadMemoire)
	upload.Post("/avatar", uploadHandler.UploadAvatar)

	admin := api.Group("/admin", middleware.AuthRequired(cfg), middleware.RequireRole("admin"))
	admin.Get("/dashboard", adminHandler.Dashboard)
	admin.Get("/accounts/users", adminHandler.ListUsers)
	admin.Post("/accounts/users", adminHandler.CreateUser)
	admin.Get("/accounts/users/:id", adminHandler.GetUser)
	admin.Patch("/accounts/users/:id", adminHandler.UpdateUser)
	admin.Post("/accounts/users/:id/action", adminHandler.UserAction)
	admin.Post("/accounts/users/import-csv", adminHandler.ImportUsersCSV)
	admin.Get("/accounts/companies", adminHandler.ListCompanies)
	admin.Post("/accounts/companies/:id/action", adminHandler.CompanyAction)
	admin.Get("/reports", adminHandler.ListReports)
	admin.Post("/reports/:id/action", adminHandler.ReportAction)
	admin.Get("/subjects", adminHandler.ListSubjects)
	admin.Get("/subjects/:id", adminHandler.GetSubject)
	admin.Post("/subjects/:id/action", adminHandler.SubjectAction)
	admin.Get("/pfe", adminHandler.ListAssignments)
	admin.Get("/pfe/:id", adminHandler.GetAssignment)
	admin.Get("/defenses", adminHandler.ListDefenses)
	admin.Post("/defenses", adminHandler.CreateDefense)
	admin.Get("/defenses/recommend-jury", adminHandler.RecommendJury)
	admin.Get("/defenses/:id", adminHandler.GetDefense)
	admin.Post("/defenses/:id/submit-grade", adminHandler.SubmitGrade)
	admin.Post("/defenses/:id/resolve-grade", adminHandler.ResolveGrade)
	admin.Post("/defenses/:id/confirm-jury", adminHandler.ConfirmJury)
	admin.Post("/defenses/:id/decline-jury", adminHandler.DeclineJury)
	admin.Get("/settings/deadlines", adminHandler.ListDeadlines)
	admin.Post("/settings/deadlines", adminHandler.UpdateDeadlines)
	admin.Get("/settings/specialities", adminHandler.ListSpecialities)
	admin.Post("/settings/specialities", adminHandler.CreateSpeciality)
	admin.Delete("/settings/specialities/:id", adminHandler.DeleteSpeciality)
	admin.Get("/settings/domains", adminHandler.ListDomains)
	admin.Post("/settings/domains", adminHandler.CreateDomain)
	admin.Delete("/settings/domains/:id", adminHandler.DeleteDomain)
	admin.Get("/settings/promotions", adminHandler.ListPromotions)
	admin.Post("/settings/promotions", adminHandler.CreatePromotion)
	admin.Delete("/settings/promotions/:id", adminHandler.DeletePromotion)
	admin.Get("/settings/academic-years", adminHandler.ListAcademicYears)
	admin.Post("/settings/academic-years", adminHandler.CreateAcademicYear)
	admin.Post("/settings/academic-years/:id/close", adminHandler.CloseAcademicYear)
	admin.Get("/statistics", adminHandler.Statistics)
	admin.Get("/audit-log", adminHandler.AuditLog)
	admin.Get("/exports/affectations", adminHandler.ExportAffectations)
	admin.Get("/exports/plannings", adminHandler.ExportPlannings)
	admin.Get("/exports/statistiques", adminHandler.ExportStatistics)

	teacher := api.Group("/teacher", middleware.AuthRequired(cfg), middleware.RequireRole("teacher", "admin"))
	teacher.Get("/dashboard", teacherHandler.Dashboard)
	teacher.Get("/proposed-subjects", teacherHandler.ListProposedSubjects)
	teacher.Post("/proposed-subjects", teacherHandler.CreateProposedSubject)
	teacher.Get("/proposed-subjects/:id", teacherHandler.GetProposedSubject)
	teacher.Patch("/proposed-subjects/:id", teacherHandler.UpdateProposedSubject)
	teacher.Get("/proposed-subjects/:id/candidats", teacherHandler.ListCandidats)
	teacher.Post("/proposed-subjects/:id/candidats", teacherHandler.AcceptCandidat)
	teacher.Get("/subjects-to-validate", teacherHandler.ListSubjectsToValidate)
	teacher.Get("/subjects-to-validate/:id", teacherHandler.GetSubjectToValidate)
	teacher.Post("/subjects-to-validate/:id", teacherHandler.ValidateSubject)
	teacher.Get("/supervised-pfes", teacherHandler.ListSupervisedPFEs)
	teacher.Get("/supervised-pfes/:id", teacherHandler.GetSupervisedPFE)
	teacher.Post("/supervised-pfes/:id/meetings", teacherHandler.AddMeeting)
	teacher.Post("/supervised-pfes/:id/evaluation", teacherHandler.SubmitEvaluation)
	teacher.Get("/jury-duties", teacherHandler.ListJuryDuties)
	teacher.Get("/jury-duties/:id", teacherHandler.GetJuryDuty)
	teacher.Post("/availability", teacherHandler.UpdateAvailability)
	teacher.Get("/notifications", teacherHandler.ListNotifications)

	student := api.Group("/student", middleware.AuthRequired(cfg), middleware.RequireRole("student"))
	student.Get("/dashboard", studentHandler.Dashboard)
	student.Get("/catalogue", studentHandler.ListCatalogue)
	student.Get("/catalogue/:id", studentHandler.GetCatalogueSubject)
	student.Get("/wishes", studentHandler.ListWishes)
	student.Post("/wishes", studentHandler.CreateWish)
	student.Delete("/wishes/:id", studentHandler.DeleteWish)
	student.Get("/my-pfe", studentHandler.GetMyPFE)
	student.Get("/my-pfe/meetings", studentHandler.ListMyMeetings)
	student.Post("/my-pfe/meetings", studentHandler.AddMyMeeting)
	student.Post("/my-pfe/memoire", studentHandler.SubmitMemoire)
	student.Get("/soutenance", studentHandler.GetSoutenance)
	student.Get("/notifications", studentHandler.ListNotifications)

	company := api.Group("/company", middleware.AuthRequired(cfg), middleware.RequireRole("company"))
	company.Get("/dashboard", companyHandler.Dashboard)
	company.Get("/subjects", companyHandler.ListSubjects)
	company.Post("/subjects", companyHandler.CreateSubject)
	company.Get("/subjects/:id", companyHandler.GetSubject)
	company.Patch("/subjects/:id", companyHandler.UpdateSubject)
	company.Get("/subjects/:id/candidats", companyHandler.ListCandidats)
	company.Post("/subjects/:id/candidats", companyHandler.AcceptCandidat)
	company.Get("/supervised-pfes", companyHandler.ListSupervisedPFEs)
	company.Get("/supervised-pfes/:id", companyHandler.GetSupervisedPFE)
	company.Post("/supervised-pfes/:id/meetings", companyHandler.AddMeeting)
	company.Post("/supervised-pfes/:id/evaluation", companyHandler.SubmitEvaluation)
	company.Get("/reports", companyHandler.ListReports)
	company.Post("/reports", companyHandler.CreateReport)
	company.Get("/notifications", companyHandler.ListNotifications)

	notifs := api.Group("/notifications", middleware.AuthRequired(cfg))
	notifs.Get("/", func(c fiber.Ctx) error {
		role := middleware.GetRole(c)
		switch role {
		case "admin", "teacher":
			return teacherHandler.ListNotifications(c)
		case "student":
			return studentHandler.ListNotifications(c)
		case "company":
			return companyHandler.ListNotifications(c)
		default:
			return c.Status(403).JSON(map[string]any{"success": false, "error": "Rôle inconnu"})
		}
	})
	notifs.Post("/:id/read", func(c fiber.Ctx) error {
		return c.JSON(map[string]any{"success": true, "data": map[string]string{"message": "Notification marquée comme lue"}})
	})
	notifs.Post("/read-all", func(c fiber.Ctx) error {
		return c.JSON(map[string]any{"success": true, "data": map[string]string{"message": "Toutes les notifications lues"}})
	})

	adminToken := generateToken(cfg.JWTSecret, SeedAdminID, "admin")

	return &TestHelper{
		App:   app,
		DB:    db,
		Cfg:   cfg,
		Admin: adminToken,
	}
}

func (h *TestHelper) Close() {
	h.DB.Close()
}

func (h *TestHelper) AuthHeader(profileID int64, role string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + generateToken(h.Cfg.JWTSecret, profileID, role),
	}
}

func (h *TestHelper) AuthToken(profileID int64, role string) string {
	return generateToken(h.Cfg.JWTSecret, profileID, role)
}

func generateToken(secret string, profileID int64, role string) string {
	claims := jwt.MapClaims{
		"sub":  profileID,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func ParseResponse(resp *http.Response) (map[string]any, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture body: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("erreur unmarshal JSON: %w (body: %s)", err, string(body))
	}
	return result, nil
}

func MustParseResponse(resp *http.Response) map[string]any {
	result, err := ParseResponse(resp)
	if err != nil {
		panic(err)
	}
	return result
}

func AssertSuccess(t TestingT, result map[string]any) {
	t.Helper()
	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Fatalf("❌ Échec: success=false, response: %+v", result)
	}
}

func AssertError(t TestingT, result map[string]any) {
	t.Helper()
	success, ok := result["success"].(bool)
	if !ok || success {
		t.Fatalf("❌ Échec: attendu erreur mais success=true, response: %+v", result)
	}
}

func AssertErrorContains(t TestingT, result map[string]any, expected string) {
	t.Helper()
	AssertError(t, result)
	errMsg, ok := result["error"].(string)
	if !ok {
		t.Fatalf("❌ Échec: champ error manquant ou non string, response: %+v", result)
	}
	if !strings.Contains(errMsg, expected) {
		t.Fatalf("❌ Échec: erreur attendue contenant %q, obtenu %q", expected, errMsg)
	}
}

type TestingT interface {
	Fatalf(format string, args ...any)
	Helper()
}

func runTestMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role TEXT NOT NULL CHECK(role IN ('admin','teacher','student','company')),
			full_name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			avatar_url TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS domains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS departments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS specialities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			code TEXT NOT NULL UNIQUE,
			year_type TEXT NOT NULL CHECK(year_type IN ('licence','master')),
			department_id INTEGER REFERENCES departments(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS academic_years (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL CHECK(status IN ('active','cloturee')),
			submission_open_at DATETIME,
			submission_close_at DATETIME,
			max_wishes INTEGER DEFAULT 5,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS promotions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT NOT NULL,
			academic_year_id INTEGER NOT NULL REFERENCES academic_years(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS teachers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id INTEGER UNIQUE NOT NULL REFERENCES profiles(id),
			grade TEXT NOT NULL CHECK(grade IN ('assistant','mab','maa','mcb','mca','professeur')),
			department_id INTEGER REFERENCES departments(id),
			availability_status TEXT DEFAULT 'disponible' CHECK(availability_status IN ('disponible','indisponible','indisponible_jusqu_au')),
			unavailable_until DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS teacher_domains (
			teacher_id INTEGER NOT NULL REFERENCES teachers(id),
			domain_id INTEGER NOT NULL REFERENCES domains(id),
			PRIMARY KEY (teacher_id, domain_id)
		)`,
		`CREATE TABLE IF NOT EXISTS students (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id INTEGER UNIQUE NOT NULL REFERENCES profiles(id),
			student_number TEXT UNIQUE NOT NULL,
			speciality_id INTEGER REFERENCES specialities(id),
			level TEXT,
			promotion_id INTEGER REFERENCES promotions(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS companies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id INTEGER UNIQUE NOT NULL REFERENCES profiles(id),
			company_name TEXT NOT NULL,
			sector TEXT,
			description TEXT,
			logo_url TEXT DEFAULT '',
			contact_email TEXT,
			contact_phone TEXT,
			website TEXT,
			is_verified BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pfe_subjects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT,
			group_type TEXT DEFAULT 'binome' CHECK(group_type IN ('monome','binome','trinome')),
			proposer_id INTEGER NOT NULL REFERENCES profiles(id),
			proposer_role TEXT NOT NULL CHECK(proposer_role IN ('teacher','company')),
			company_id INTEGER REFERENCES companies(id),
			academic_year_id INTEGER NOT NULL REFERENCES academic_years(id),
			validator1_id INTEGER REFERENCES teachers(id),
			validator2_id INTEGER REFERENCES teachers(id),
			validator1_decision TEXT CHECK(validator1_decision IN ('valide','accepte_sous_reserve','refuse')),
			validator2_decision TEXT CHECK(validator2_decision IN ('valide','accepte_sous_reserve','refuse')),
			validator1_comment TEXT,
			validator2_comment TEXT,
			status TEXT DEFAULT 'en_attente' CHECK(status IN ('en_attente','valide','accepte_sous_reserve','refuse','expire')),
			co_supervisor_id INTEGER REFERENCES teachers(id),
			pre_assigned_student_ids TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS subject_domains (
			subject_id INTEGER NOT NULL REFERENCES pfe_subjects(id),
			domain_id INTEGER NOT NULL REFERENCES domains(id),
			PRIMARY KEY (subject_id, domain_id)
		)`,
		`CREATE TABLE IF NOT EXISTS wishes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			student_id INTEGER NOT NULL REFERENCES students(id),
			subject_id INTEGER NOT NULL REFERENCES pfe_subjects(id),
			academic_year_id INTEGER NOT NULL REFERENCES academic_years(id),
			status TEXT DEFAULT 'en_attente' CHECK(status IN ('en_attente','accepte','refuse')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pfe_assignments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pfe_code TEXT UNIQUE NOT NULL,
			subject_id INTEGER NOT NULL REFERENCES pfe_subjects(id),
			academic_year_id INTEGER NOT NULL REFERENCES academic_years(id),
			student_id INTEGER NOT NULL REFERENCES students(id),
			student2_id INTEGER REFERENCES students(id),
			student3_id INTEGER REFERENCES students(id),
			supervisor_id INTEGER NOT NULL REFERENCES teachers(id),
			co_supervisor_id INTEGER REFERENCES teachers(id),
			memoire_url TEXT,
			status TEXT DEFAULT 'en_cours' CHECK(status IN ('en_cours','memoire_soumis','soutenance_planifiee','valide','refuse')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pfe_progress_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			assignment_id INTEGER NOT NULL REFERENCES pfe_assignments(id),
			meeting_date DATETIME NOT NULL,
			duration INTEGER NOT NULL,
			meeting_type TEXT NOT NULL CHECK(meeting_type IN ('presentiel','visio')),
			topics TEXT,
			status TEXT DEFAULT 'en_cours' CHECK(status IN ('en_cours','termine')),
			observation TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS defense_juries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			assignment_id INTEGER NOT NULL REFERENCES pfe_assignments(id),
			president_id INTEGER NOT NULL REFERENCES teachers(id),
			member_id INTEGER NOT NULL REFERENCES teachers(id),
			president_confirmed BOOLEAN DEFAULT 0,
			member_confirmed BOOLEAN DEFAULT 0,
			president_wants_printed BOOLEAN DEFAULT 0,
			member_wants_printed BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS defenses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			assignment_id INTEGER NOT NULL REFERENCES pfe_assignments(id),
			jury_id INTEGER NOT NULL REFERENCES defense_juries(id),
			scheduled_at DATETIME,
			room TEXT,
			defense_deadline DATETIME,
			status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled','done','postponed')),
			result TEXT CHECK(result IN ('admitted','corrections_required','not_admitted')),
			final_grade REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS jury_grades (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			defense_id INTEGER NOT NULL REFERENCES defenses(id),
			jury_member_id INTEGER NOT NULL REFERENCES teachers(id),
			criterion1 REAL,
			criterion2 REAL,
			criterion3 REAL,
			criterion4 REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS supervisor_evaluations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pfe_assignment_id INTEGER UNIQUE NOT NULL REFERENCES pfe_assignments(id),
			evaluator_id INTEGER NOT NULL REFERENCES teachers(id),
			criterion5 REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS company_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id INTEGER NOT NULL REFERENCES companies(id),
			submitted_by INTEGER NOT NULL,
			correction_type TEXT NOT NULL,
			description TEXT,
			requested_value TEXT,
			status TEXT DEFAULT 'en_attente' CHECK(status IN ('en_attente','resolu','rejete')),
			resolved_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			recipient_id INTEGER NOT NULL REFERENCES profiles(id),
			type TEXT NOT NULL,
			payload TEXT DEFAULT '{}',
			read_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			actor_id INTEGER NOT NULL REFERENCES profiles(id),
			action TEXT NOT NULL,
			entity TEXT NOT NULL,
			entity_id INTEGER,
			metadata TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration test échouée: %w\nSQL: %s", err, m[:50])
		}
	}
	return nil
}

func runTestSeed(db *sql.DB) error {

	seeds := []string{

		`INSERT OR IGNORE INTO domains (id, name) VALUES (1, 'Intelligence Artificielle')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (2, 'Développement Web')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (3, 'Réseaux et Sécurité')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (4, 'Data Science')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (5, 'Systèmes Embarqués')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (6, 'Développement Mobile')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (7, 'Cloud Computing')`,
		`INSERT OR IGNORE INTO domains (id, name) VALUES (8, 'Bio-Informatique')`,

		`INSERT OR IGNORE INTO departments (id, name) VALUES (1, 'Informatique')`,
		`INSERT OR IGNORE INTO departments (id, name) VALUES (2, 'Chimie')`,

		`INSERT OR IGNORE INTO specialities (id, name, code, year_type, department_id) VALUES (1, 'ISIL', 'ISIL', 'master', 1)`,
		`INSERT OR IGNORE INTO specialities (id, name, code, year_type, department_id) VALUES (2, 'Chimie', 'CHIM', 'licence', 2)`,
		`INSERT OR IGNORE INTO specialities (id, name, code, year_type, department_id) VALUES (3, 'Électrotechnique', 'ELEC', 'master', 1)`,

		`INSERT OR IGNORE INTO academic_years (id, label, status) VALUES (1, '2023-2024', 'cloturee')`,
		`INSERT OR IGNORE INTO academic_years (id, label, status, submission_open_at, submission_close_at, max_wishes)
		 VALUES (2, '2024-2025', 'active', datetime('now', '-30 days'), datetime('now', '+30 days'), 5)`,

		`INSERT OR IGNORE INTO promotions (id, label, academic_year_id) VALUES (1, 'ISIL 2024-2025', 2)`,
		`INSERT OR IGNORE INTO promotions (id, label, academic_year_id) VALUES (2, 'CHIM 2024-2025', 2)`,

		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (1, 'admin', 'Admin Test', 'admin@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (2, 'teacher', 'Dr. ISIL Teacher', 'teacher.isil@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (3, 'teacher', 'Pr. ISIL Validator', 'teacher.isil2@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (4, 'teacher', 'Dr. CHIM Teacher', 'teacher.chim@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (5, 'student', 'Étudiant ISIL 1', 'student.isil1@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (6, 'student', 'Étudiant ISIL 2', 'student.isil2@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (7, 'student', 'Étudiant CHIM 1', 'student.chim1@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (8, 'student', 'Étudiant ISIL 4', 'student.isil4@test.dz', 1)`,
		`INSERT OR IGNORE INTO profiles (id, role, full_name, email, is_active) VALUES (9, 'company', 'TechCorp Algérie', 'contact@techcorp.dz', 1)`,

		`INSERT OR IGNORE INTO teachers (id, profile_id, grade, department_id, availability_status) VALUES (1, 2, 'mca', 1, 'disponible')`,
		`INSERT OR IGNORE INTO teachers (id, profile_id, grade, department_id, availability_status) VALUES (2, 3, 'professeur', 1, 'disponible')`,
		`INSERT OR IGNORE INTO teachers (id, profile_id, grade, department_id, availability_status) VALUES (3, 4, 'mcb', 2, 'disponible')`,

		`INSERT OR IGNORE INTO teacher_domains (teacher_id, domain_id) VALUES (1, 1)`,
		`INSERT OR IGNORE INTO teacher_domains (teacher_id, domain_id) VALUES (1, 2)`,
		`INSERT OR IGNORE INTO teacher_domains (teacher_id, domain_id) VALUES (2, 1)`,
		`INSERT OR IGNORE INTO teacher_domains (teacher_id, domain_id) VALUES (3, 4)`,

		`INSERT OR IGNORE INTO students (id, profile_id, student_number, speciality_id, level, promotion_id) VALUES (1, 5, '2024001', 1, 'M2', 1)`,
		`INSERT OR IGNORE INTO students (id, profile_id, student_number, speciality_id, level, promotion_id) VALUES (2, 6, '2024002', 1, 'M2', 1)`,
		`INSERT OR IGNORE INTO students (id, profile_id, student_number, speciality_id, level, promotion_id) VALUES (3, 7, '2024003', 2, 'L3', 2)`,
		`INSERT OR IGNORE INTO students (id, profile_id, student_number, speciality_id, level, promotion_id) VALUES (4, 8, '2024004', 1, 'M2', 1)`,

		`INSERT OR IGNORE INTO companies (id, profile_id, company_name, sector, description, is_verified) VALUES (1, 9, 'TechCorp Algérie', 'Technologie', 'Entreprise tech', 1)`,

		`INSERT OR IGNORE INTO pfe_subjects (id, title, description, group_type, proposer_id, proposer_role, academic_year_id, status)
		 VALUES (1, 'IA pour la santé', 'Sujet IA santé', 'binome', 2, 'teacher', 2, 'en_attente')`,
		`INSERT OR IGNORE INTO pfe_subjects (id, title, description, group_type, proposer_id, proposer_role, academic_year_id, status)
		 VALUES (2, 'Web App Sécurité', 'Sujet web sécurité', 'monome', 2, 'teacher', 2, 'valide')`,
		`INSERT OR IGNORE INTO pfe_subjects (id, title, description, group_type, proposer_id, proposer_role, academic_year_id, status,
		    validator1_id, validator2_id, validator1_decision, validator2_decision)
		 VALUES (3, 'Cloud Computing Avancé', 'Sujet cloud', 'binome', 2, 'teacher', 2, 'valide',
		    2, 3, 'valide', 'valide')`,
		`INSERT OR IGNORE INTO pfe_subjects (id, title, description, group_type, proposer_id, proposer_role, academic_year_id, status,
		    validator1_id, validator1_decision)
		 VALUES (4, 'Data Mining', 'Sujet data', 'binome', 4, 'teacher', 2, 'accepte_sous_reserve',
		    2, 'accepte_sous_reserve')`,
		`INSERT OR IGNORE INTO pfe_subjects (id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id, status)
		 VALUES (5, 'IoT Industriel', 'Sujet IoT', 'trinome', 9, 'company', 1, 2, 'valide')`,
		`INSERT OR IGNORE INTO pfe_subjects (id, title, description, group_type, proposer_id, proposer_role, academic_year_id, status)
		 VALUES (6, 'Blockchain', 'Sujet blockchain', 'monome', 2, 'teacher', 2, 'refuse')`,

		`INSERT OR IGNORE INTO subject_domains (subject_id, domain_id) VALUES (1, 1)`,
		`INSERT OR IGNORE INTO subject_domains (subject_id, domain_id) VALUES (2, 2)`,
		`INSERT OR IGNORE INTO subject_domains (subject_id, domain_id) VALUES (3, 7)`,

		`INSERT OR IGNORE INTO wishes (id, student_id, subject_id, academic_year_id, status) VALUES (1, 1, 2, 2, 'en_attente')`,
		`INSERT OR IGNORE INTO wishes (id, student_id, subject_id, academic_year_id, status) VALUES (2, 1, 3, 2, 'en_attente')`,
		`INSERT OR IGNORE INTO wishes (id, student_id, subject_id, academic_year_id, status) VALUES (3, 2, 3, 2, 'accepte')`,

		`INSERT OR IGNORE INTO pfe_assignments (id, pfe_code, subject_id, academic_year_id, student_id, student2_id, supervisor_id, status)
		 VALUES (1, 'PFE-ISIL-2025-001', 3, 2, 1, 2, 1, 'en_cours')`,
		`INSERT OR IGNORE INTO pfe_assignments (id, pfe_code, subject_id, academic_year_id, student_id, supervisor_id, status)
		 VALUES (2, 'PFE-ISIL-2025-002', 5, 2, 4, 1, 'en_cours')`,

		`INSERT OR IGNORE INTO pfe_progress_reports (id, assignment_id, meeting_date, duration, meeting_type, topics, status)
		 VALUES (1, 1, datetime('now', '-14 days'), 60, 'presentiel', 'Introduction, planification', 'termine')`,
		`INSERT OR IGNORE INTO pfe_progress_reports (id, assignment_id, meeting_date, duration, meeting_type, topics, status)
		 VALUES (2, 1, datetime('now', '-7 days'), 45, 'visio', 'État d''avancement', 'termine')`,

		`INSERT OR IGNORE INTO supervisor_evaluations (id, pfe_assignment_id, evaluator_id, criterion5) VALUES (1, 1, 1, 3.5)`,

		`INSERT OR IGNORE INTO defense_juries (id, assignment_id, president_id, member_id, president_confirmed, member_confirmed)
		 VALUES (1, 1, 2, 3, 1, 1)`,

		`INSERT OR IGNORE INTO defenses (id, assignment_id, jury_id, scheduled_at, room, status)
		 VALUES (1, 1, 1, datetime('now', '+14 days'), 'Salle A', 'scheduled')`,

		`INSERT OR IGNORE INTO notifications (id, recipient_id, type, payload) VALUES (1, 1, 'nouveau_sujet', '{"subject_id":1}')`,
		`INSERT OR IGNORE INTO notifications (id, recipient_id, type, payload) VALUES (2, 2, 'sujet_valide', '{"subject_id":3}')`,
	}

	for _, s := range seeds {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("seed test échoué: %w\nSQL: %s", err, s[:50])
		}
	}
	return nil
}

func CleanDB(db *sql.DB) error {
	tables := []string{
		"audit_logs", "notifications", "company_reports", "supervisor_evaluations",
		"jury_grades", "defenses", "defense_juries", "pfe_progress_reports",
		"pfe_assignments", "wishes", "subject_domains", "pfe_subjects",
		"companies", "students", "teacher_domains", "teachers",
		"promotions", "academic_years", "specialities", "domains", "profiles",
	}
	for _, t := range tables {
		if _, err := db.Exec("DELETE FROM " + t); err != nil {
			return fmt.Errorf("erreur nettoyage %s: %w", t, err)
		}
	}
	return nil
}
