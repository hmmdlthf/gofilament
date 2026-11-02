package filament

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/golaravel/pkg/translation"
)

// RequiredTranslationKeys defines all mandatory translation keys that every supported
// admin panel locale must provide. If a locale is registered or activated without
// fulfilling these keys, GoFilament's validation system reports an explicit error.
var RequiredTranslationKeys = []string{
	// Navigation & Shell
	"dashboard",
	"entities",
	"resources",
	"account",
	"settings",
	"orders",
	"billing",
	"security",
	"devices",
	"telemetry",
	"tickets",
	"firmware",
	"operators",
	"analytics",
	"theme",
	"language",
	"sign_out",
	"search",
	"add",
	"save",
	"save_changes",
	"cancel",
	"delete",
	"edit",
	"close",
	"apply",
	"back",
	"status",
	"active_session",
	"refresh",
	"system",
	"control_center",

	// Authentication / Sign In
	"sign_in",
	"sign_in_title",
	"sign_in_subtitle",
	"username_or_email",
	"password",
	"remember_me",
	"enter_credentials",
	"quick_demo_roles",
	"enclave_secured",
	"switch_to_arabic",
	"switch_to_english",

	// Records & Data Table
	"search_records",
	"filters",
	"filter",
	"clear_all",
	"select_all",
	"deselect_all",
	"bulk_actions",
	"bulk_delete",
	"export_selected",
	"clear_selection",
	"no_records_found",
	"rows_per_page",
	"page",
	"of",
	"actions",
	"create_record",
	"edit_record",
	"delete_record",
	"delete_selected",
	"delete_confirm_title",
	"delete_records_title",
	"delete_confirm",
	"delete_records_confirm",
	"record_created",
	"record_updated",
	"record_deleted",
	"records_deleted",
	"operation_failed",

	// Relations & Nested Managers
	"show_relations",
	"hide_relations",
	"toggle_relations",
	"relations",
	"attributes",
	"manage",
	"related_records",

	// Sorting & Table Controls
	"sort_by",
	"clear_sort",
	"ascending",
	"descending",

	// Pagination & Infinite Scroll
	"pagination",
	"infinite_scroll",
	"previous_page",
	"next_page",
	"first_page",
	"last_page",
	"showing_records",
	"to",
	"load_more",
	"loading_more",
	"all_records_loaded",

	// Dashboard & Metrics
	"operational_dashboard",
	"admin_dashboard",
	"performance_overview",
	"dashboard_subtitle",
	"refresh_dashboard",
	"active_entities",
	"system_status",
	"operational",
	"multi_tenancy",
	"quick_access_entities",

	// Language, Direction & Theme
	"language_picker",
	"direction",
	"rtl",
	"ltr",
	"english",
	"arabic",
	"dark_mode",
	"light_mode",
	"theme_active_dark",
	"theme_active_light",
}

var defaultAdminTranslations = map[string]map[string]string{
	"en": {
		"dashboard":              "Dashboard",
		"entities":               "Entities",
		"resources":              "Resources",
		"account":                "Account",
		"settings":               "Settings & Preferences",
		"orders":                 "Orders",
		"billing":                "Billing",
		"security":               "Security",
		"biometrics":             "Biometrics",
		"devices":                "Devices",
		"telemetry":              "Telemetry",
		"tickets":                "Tickets",
		"firmware":               "Firmware",
		"operators":              "Team",
		"analytics":              "Metrics",
		"theme":                  "Theme",
		"language":               "Language",
		"sign_out":               "Sign Out",
		"search":                 "Search",
		"add":                    "Add",
		"save":                   "Save",
		"cancel":                 "Cancel",
		"delete":                 "Delete",
		"edit":                   "Edit",
		"status":                 "Status",
		"active_session":         "Active Session",
		"dark_mode":              "Dark Mode",
		"light_mode":             "Light Mode",
		"theme_active_dark":      "Dark theme active",
		"theme_active_light":     "Light theme active",
		"refresh":                "Refresh",
		"system":                 "System",
		"control_center":         "Control Center",
		"sign_in":                "Sign In",
		"sign_in_title":          "Sign In to Your Workspace",
		"sign_in_subtitle":       "Enter your authorized credentials to access this administration panel.",
		"username_or_email":      "Username or Email",
		"password":               "Password",
		"remember_me":            "Remember me",
		"enter_credentials":      "Please enter both username and password",
		"quick_demo_roles":       "QUICK DEMO ROLES (CLICK TO AUTOFILL)",
		"enclave_secured":        "Enclave Secured • GoFilament & GoLaravel Engine",
		"switch_to_arabic":       "🇸🇦 العربية (RTL)",
		"switch_to_english":      "🇺🇸 English (LTR)",
		"search_records":         "Search records...",
		"filters":                "Filters",
		"filter":                 "Filter",
		"clear_all":              "Clear All",
		"select_all":             "Select All",
		"deselect_all":           "Deselect All",
		"bulk_actions":           "Bulk Actions",
		"bulk_delete":            "Bulk Delete",
		"export_selected":        "Export Selected",
		"clear_selection":        "Clear Selection",
		"no_records_found":       "No records found",
		"rows_per_page":          "Rows per page",
		"page":                   "Page",
		"of":                     "of",
		"actions":                "Actions",
		"create_record":          "Create Record",
		"edit_record":            "Edit Record",
		"delete_record":          "Delete Record",
		"delete_selected":        "Delete Selected",
		"delete_confirm_title":   "Delete Record?",
		"delete_records_title":   "Delete Records?",
		"delete_confirm":         "Are you sure you want to delete this record? This action cannot be undone.",
		"delete_records_confirm": "Are you sure you want to delete these records? This action cannot be undone.",
		"record_created":         "Record created successfully",
		"record_updated":         "Record updated successfully",
		"record_deleted":         "Record deleted successfully",
		"records_deleted":        "Records deleted successfully",
		"operation_failed":       "Operation failed. Please try again.",
		"show_relations":         "Show Related Entities",
		"hide_relations":         "Hide Related Entities",
		"toggle_relations":       "Toggle Relations",
		"relations":              "Relations",
		"attributes":             "Attributes",
		"manage":                 "Manage",
		"related_records":        "Related Records",
		"back":                   "Back",
		"operational_dashboard":  "Operational Dashboard",
		"admin_dashboard":        "Admin Dashboard",
		"performance_overview":   "Performance Overview",
		"dashboard_subtitle":     "Live operational metrics, telematics & hardware analytics",
		"refresh_dashboard":      "Refresh Dashboard",
		"active_entities":        "Active Entities",
		"system_status":          "System Status",
		"operational":            "Operational",
		"multi_tenancy":          "Multi-Tenancy",
		"quick_access_entities":  "Quick Access Entities",
		"language_picker":        "Select Language & Direction",
		"direction":              "Direction",
		"rtl":                    "Right-to-Left (RTL)",
		"ltr":                    "Left-to-Right (LTR)",
		"english":                "English",
		"arabic":                 "العربية",
		"close":                  "Close",
		"apply":                  "Apply",
		"save_changes":           "Save Changes",
		"sort_by":                "Sort By",
		"clear_sort":             "Clear Sort",
		"ascending":              "Ascending",
		"descending":             "Descending",
		"pagination":             "Pagination",
		"infinite_scroll":        "Infinite Scroll",
		"previous_page":          "Previous",
		"next_page":              "Next",
		"first_page":             "First",
		"last_page":              "Last",
		"showing_records":        "Showing",
		"to":                     "to",
		"load_more":              "Load More",
		"loading_more":           "Loading more records...",
		"all_records_loaded":     "All records loaded",
	},
	"ar": {
		"dashboard":              "لوحة التحكم",
		"entities":               "الكيانات",
		"resources":              "الموارد",
		"account":                "الحساب",
		"settings":               "الإعدادات والتفضيلات",
		"orders":                 "الطلبات",
		"billing":                "الفواتير",
		"security":               "الأمان",
		"biometrics":             "القياسات الحيوية",
		"devices":                "الأجهزة",
		"telemetry":              "القياس عن بُعد",
		"tickets":                "التذاكر",
		"firmware":               "البرامج الثابتة",
		"operators":              "فريق العمل",
		"analytics":              "المؤشرات",
		"theme":                  "المظهر",
		"language":               "اللغة",
		"sign_out":               "تسجيل الخروج",
		"search":                 "بحث",
		"add":                    "إضافة",
		"save":                   "حفظ",
		"cancel":                 "إلغاء",
		"delete":                 "حذف",
		"edit":                   "تعديل",
		"status":                 "الحالة",
		"active_session":         "جلسة نشطة",
		"dark_mode":              "الوضع الداكن",
		"light_mode":             "الوضع الفاتح",
		"theme_active_dark":      "الوضع الداكن مفعّل",
		"theme_active_light":     "الوضع الفاتح مفعّل",
		"refresh":                "تحديث",
		"system":                 "النظام",
		"control_center":         "مركز التحكم",
		"sign_in":                "تسجيل الدخول",
		"sign_in_title":          "تسجيل الدخول إلى مساحة العمل",
		"sign_in_subtitle":       "أدخل بيانات الاعتماد المصرح بها للوصول إلى لوحة الإدارة.",
		"username_or_email":      "اسم المستخدم أو البريد الإلكتروني",
		"password":               "كلمة المرور",
		"remember_me":            "تذكرني",
		"enter_credentials":      "يرجى إدخال اسم المستخدم وكلمة المرور معاً",
		"quick_demo_roles":       "أدوار العرض السريع (انقر للتعبئة التلقائية)",
		"enclave_secured":        "محمي بالجيب الأمني • محرك GoFilament و GoLaravel",
		"switch_to_arabic":       "🇸🇦 العربية (RTL)",
		"switch_to_english":      "🇺🇸 English (LTR)",
		"search_records":         "البحث في السجلات...",
		"filters":                "عوامل التصفية",
		"filter":                 "تصفية",
		"clear_all":              "مسح الكل",
		"select_all":             "تحديد الكل",
		"deselect_all":           "إلغاء تحديد الكل",
		"bulk_actions":           "إجراءات جماعية",
		"bulk_delete":            "حذف جماعي",
		"export_selected":        "تصدير المحدد",
		"clear_selection":        "مسح التحديد",
		"no_records_found":       "لم يتم العثور على سجلات",
		"rows_per_page":          "الصفوف لكل صفحة",
		"page":                   "صفحة",
		"of":                     "من",
		"actions":                "إجراءات",
		"create_record":          "إنشاء سجل جديد",
		"edit_record":            "تعديل السجل",
		"delete_record":          "حذف السجل",
		"delete_selected":        "حذف المحدد",
		"delete_confirm_title":   "حذف السجل؟",
		"delete_records_title":   "حذف السجلات؟",
		"delete_confirm":         "هل أنت متأكد من حذف هذا السجل؟ لا يمكن التراجع عن هذا الإجراء.",
		"delete_records_confirm": "هل أنت متأكد من حذف السجلات المحددة؟ لا يمكن التراجع عن هذا الإجراء.",
		"record_created":         "تم إنشاء السجل بنجاح",
		"record_updated":         "تم تحديث السجل بنجاح",
		"record_deleted":         "تم حذف السجل بنجاح",
		"records_deleted":        "تم حذف السجلات بنجاح",
		"operation_failed":       "فشلت العملية. يرجى المحاولة مرة أخرى.",
		"show_relations":         "عرض الكيانات المرتبطة",
		"hide_relations":         "إخفاء الكيانات المرتبطة",
		"toggle_relations":       "تبديل الكيانات المرتبطة",
		"relations":              "العلاقات",
		"attributes":             "الخصائص",
		"manage":                 "إدارة",
		"related_records":        "الكيانات المرتبطة",
		"back":                   "رجوع",
		"operational_dashboard":  "لوحة العمليات التشغيلية",
		"admin_dashboard":        "لوحة الإدارة",
		"performance_overview":   "نظرة عامة على الأداء",
		"dashboard_subtitle":     "مؤشرات تشغيلية حية، القياس عن بُعد وتحليلات الأجهزة",
		"refresh_dashboard":      "تحديث لوحة التحكم",
		"active_entities":        "الكيانات النشطة",
		"system_status":          "حالة النظام",
		"operational":            "يعمل بكفاءة",
		"multi_tenancy":          "تعدد المستأجرين",
		"quick_access_entities":  "الوصول السريع للكيانات",
		"language_picker":        "اختر اللغة واتجاه الواجهة",
		"direction":              "الاتجاه",
		"rtl":                    "من اليمين إلى اليسار (RTL)",
		"ltr":                    "من اليسار إلى اليمين (LTR)",
		"english":                "الإنجليزية",
		"arabic":                 "العربية",
		"close":                  "إغلاق",
		"apply":                  "تطبيق",
		"save_changes":           "حفظ التغييرات",
		"sort_by":                "الترتيب حسب",
		"clear_sort":             "إلغاء الترتيب",
		"ascending":              "تصاعدي",
		"descending":             "تنازلي",
		"pagination":             "ترقيم الصفحات",
		"infinite_scroll":        "التمرير اللانهائي",
		"previous_page":          "السابق",
		"next_page":              "التالي",
		"first_page":             "الأول",
		"last_page":              "الأخير",
		"showing_records":        "عرض",
		"to":                     "إلى",
		"load_more":              "تحميل المزيد",
		"loading_more":           "جاري تحميل المزيد من السجلات...",
		"all_records_loaded":     "تم تحميل جميع السجلات",
	},
	"es": {
		"dashboard":              "Panel de Control",
		"entities":               "Entidades",
		"resources":              "Recursos",
		"account":                "Cuenta",
		"settings":               "Configuración",
		"orders":                 "Pedidos",
		"billing":                "Facturación",
		"security":               "Seguridad",
		"biometrics":             "Biometría",
		"devices":                "Dispositivos",
		"telemetry":              "Telemetría",
		"tickets":                "Tickets",
		"firmware":               "Firmware",
		"operators":              "Equipo",
		"analytics":              "Métricas",
		"theme":                  "Tema",
		"language":               "Idioma",
		"sign_out":               "Cerrar Sesión",
		"search":                 "Buscar",
		"add":                    "Añadir",
		"save":                   "Guardar",
		"cancel":                 "Cancelar",
		"delete":                 "Eliminar",
		"edit":                   "Editar",
		"status":                 "Estado",
		"active_session":         "Sesión Activa",
		"dark_mode":              "Modo Oscuro",
		"light_mode":             "Modo Claro",
		"theme_active_dark":      "Tema oscuro activo",
		"theme_active_light":     "Tema claro activo",
		"refresh":                "Refrescar",
		"system":                 "Sistema",
		"control_center":         "Centro de Control",
		"sign_in":                "Iniciar Sesión",
		"sign_in_title":          "Iniciar Sesión en su Espacio de Trabajo",
		"sign_in_subtitle":       "Ingrese sus credenciales autorizadas para acceder a este panel de administración.",
		"username_or_email":      "Usuario o Correo Electrónico",
		"password":               "Contraseña",
		"remember_me":            "Recordarme",
		"enter_credentials":      "Por favor ingrese usuario y contraseña",
		"quick_demo_roles":       "ROLES DE DEMO RÁPIDA (CLIC PARA AUTOLLENAR)",
		"enclave_secured":        "Enclave Seguro • GoFilament & GoLaravel Engine",
		"switch_to_arabic":       "🇸🇦 العربية (RTL)",
		"switch_to_english":      "🇺🇸 English (LTR)",
		"search_records":         "Buscar registros...",
		"filters":                "Filtros",
		"filter":                 "Filtrar",
		"clear_all":              "Borrar Todo",
		"select_all":             "Seleccionar Todo",
		"deselect_all":           "Deseleccionar Todo",
		"bulk_actions":           "Acciones Masivas",
		"bulk_delete":            "Eliminación Masiva",
		"export_selected":        "Exportar Seleccionados",
		"clear_selection":        "Borrar Selección",
		"no_records_found":       "No se encontraron registros",
		"rows_per_page":          "Filas por página",
		"page":                   "Página",
		"of":                     "de",
		"actions":                "Acciones",
		"create_record":          "Crear Registro",
		"edit_record":            "Editar Registro",
		"delete_record":          "Eliminar Registro",
		"delete_selected":        "Eliminar Seleccionados",
		"delete_confirm_title":   "¿Eliminar Registro?",
		"delete_records_title":   "¿Eliminar Registros?",
		"delete_confirm":         "¿Está seguro de que desea eliminar este registro? Esta acción no se puede deshacer.",
		"delete_records_confirm": "¿Está seguro de que desea eliminar los registros seleccionados? Esta acción no se puede deshacer.",
		"record_created":         "Registro creado exitosamente",
		"record_updated":         "Registro actualizado exitosamente",
		"record_deleted":         "Registro eliminado exitosamente",
		"records_deleted":        "Registros eliminados exitosamente",
		"operation_failed":       "La operación falló. Por favor intente de nuevo.",
		"show_relations":         "Mostrar Entidades Relacionadas",
		"hide_relations":         "Ocultar Entidades Relacionadas",
		"toggle_relations":       "Alternar Relaciones",
		"relations":              "Relaciones",
		"attributes":             "Atributos",
		"manage":                 "Administrar",
		"related_records":        "Registros Relacionados",
		"back":                   "Atrás",
		"operational_dashboard":  "Panel Operativo",
		"admin_dashboard":        "Panel de Administración",
		"performance_overview":   "Resumen de Rendimiento",
		"dashboard_subtitle":     "Métricas operativas en vivo, telemática y análisis de hardware",
		"refresh_dashboard":      "Actualizar Panel",
		"active_entities":        "Entidades Activas",
		"system_status":          "Estado del Sistema",
		"operational":            "Operativo",
		"multi_tenancy":          "Multiinquilino",
		"quick_access_entities":  "Entidades de Acceso Rápido",
		"language_picker":        "Seleccionar Idioma y Dirección",
		"direction":              "Dirección",
		"rtl":                    "De Derecha a Izquierda (RTL)",
		"ltr":                    "De Izquierda a Derecha (LTR)",
		"english":                "Inglés",
		"arabic":                 "Árabe",
		"close":                  "Cerrar",
		"apply":                  "Aplicar",
		"save_changes":           "Guardar Cambios",
		"sort_by":                "Ordenar por",
		"clear_sort":             "Borrar ordenación",
		"ascending":              "Ascendente",
		"descending":             "Descendente",
		"pagination":             "Paginación",
		"infinite_scroll":        "Desplazamiento Infinito",
		"previous_page":          "Anterior",
		"next_page":              "Siguiente",
		"first_page":             "Primero",
		"last_page":              "Último",
		"showing_records":        "Mostrando",
		"to":                     "a",
		"load_more":              "Cargar más",
		"loading_more":           "Cargando más registros...",
		"all_records_loaded":     "Todos los registros cargados",
	},
	"de": {
		"dashboard":              "Übersicht",
		"entities":               "Entitäten",
		"resources":              "Ressourcen",
		"account":                "Konto",
		"settings":               "Einstellungen",
		"orders":                 "Bestellungen",
		"billing":                "Abrechnung",
		"security":               "Sicherheit",
		"biometrics":             "Biometrie",
		"devices":                "Geräte",
		"telemetry":              "Telemetrie",
		"tickets":                "Tickets",
		"firmware":               "Firmware",
		"operators":              "Team",
		"analytics":              "Metriken",
		"theme":                  "Design",
		"language":               "Sprache",
		"sign_out":               "Abmelden",
		"search":                 "Suchen",
		"add":                    "Hinzufügen",
		"save":                   "Speichern",
		"cancel":                 "Abbrechen",
		"delete":                 "Löschen",
		"edit":                   "Bearbeiten",
		"status":                 "Status",
		"active_session":         "Aktive Sitzung",
		"dark_mode":              "Dunkelmodus",
		"light_mode":             "Hellmodus",
		"theme_active_dark":      "Dunkelmodus aktiv",
		"theme_active_light":     "Hellmodus aktiv",
		"refresh":                "Aktualisieren",
		"system":                 "System",
		"control_center":         "Kontrollzentrum",
		"sign_in":                "Anmelden",
		"sign_in_title":          "Im Arbeitsbereich anmelden",
		"sign_in_subtitle":       "Geben Sie Ihre autorisierten Anmeldedaten ein, um auf das Administrationspanel zuzugreifen.",
		"username_or_email":      "Benutzername oder E-Mail",
		"password":               "Passwort",
		"remember_me":            "Angemeldet bleiben",
		"enter_credentials":      "Bitte Benutzername und Passwort eingeben",
		"quick_demo_roles":       "SCHNELLE DEMO-ROLLEN (ZUM AUTO-AUSFÜLLEN KLICKEN)",
		"enclave_secured":        "Sichere Enklave • GoFilament & GoLaravel Engine",
		"switch_to_arabic":       "🇸🇦 العربية (RTL)",
		"switch_to_english":      "🇺🇸 English (LTR)",
		"search_records":         "Datensätze durchsuchen...",
		"filters":                "Filter",
		"filter":                 "Filtern",
		"clear_all":              "Alle leeren",
		"select_all":             "Alle auswählen",
		"deselect_all":           "Auswahl aufheben",
		"bulk_actions":           "Massenaktionen",
		"bulk_delete":            "Massenlöschung",
		"export_selected":        "Ausgewählte exportieren",
		"clear_selection":        "Auswahl aufheben",
		"no_records_found":       "Keine Datensätze gefunden",
		"rows_per_page":          "Zeilen pro Seite",
		"page":                   "Seite",
		"of":                     "von",
		"actions":                "Aktionen",
		"create_record":          "Datensatz erstellen",
		"edit_record":            "Datensatz bearbeiten",
		"delete_record":          "Datensatz löschen",
		"delete_selected":        "Ausgewählte löschen",
		"delete_confirm_title":   "Datensatz löschen?",
		"delete_records_title":   "Datensätze löschen?",
		"delete_confirm":         "Sind Sie sicher, dass Sie diesen Datensatz löschen möchten? Dies kann nicht rückgängig gemacht werden.",
		"delete_records_confirm": "Sind Sie sicher, dass Sie diese ausgewählten Datensätze löschen möchten? Dies kann nicht rückgängig gemacht werden.",
		"record_created":         "Datensatz erfolgreich erstellt",
		"record_updated":         "Datensatz erfolgreich aktualisiert",
		"record_deleted":         "Datensatz erfolgreich gelöscht",
		"records_deleted":        "Datensätze erfolgreich gelöscht",
		"operation_failed":       "Vorgang fehlgeschlagen. Bitte erneut versuchen.",
		"show_relations":         "Verknüpfte Entitäten anzeigen",
		"hide_relations":         "Verknüpfte Entitäten ausblenden",
		"toggle_relations":       "Relationen umschalten",
		"relations":              "Relationen",
		"attributes":             "Attribute",
		"manage":                 "Verwalten",
		"related_records":        "Verknüpfte Datensätze",
		"back":                   "Zurück",
		"operational_dashboard":  "Betriebsübersicht",
		"admin_dashboard":        "Administrations-Dashboard",
		"performance_overview":   "Leistungsübersicht",
		"dashboard_subtitle":     "Live-Betriebskennzahlen, Telematik & Hardware-Analysen",
		"refresh_dashboard":      "Dashboard aktualisieren",
		"active_entities":        "Aktive Entitäten",
		"system_status":          "Systemstatus",
		"operational":            "Betriebsbereit",
		"multi_tenancy":          "Mandantenfähigkeit",
		"quick_access_entities":  "Schnellzugriff auf Entitäten",
		"language_picker":        "Sprache & Ausrichtung wählen",
		"direction":              "Ausrichtung",
		"rtl":                    "Rechts nach Links (RTL)",
		"ltr":                    "Links nach Rechts (LTR)",
		"english":                "Englisch",
		"arabic":                 "Arabisch",
		"close":                  "Schließen",
		"apply":                  "Anwenden",
		"save_changes":           "Änderungen speichern",
		"sort_by":                "Sortieren nach",
		"clear_sort":             "Sortierung aufheben",
		"ascending":              "Aufsteigend",
		"descending":             "Absteigend",
		"pagination":             "Seitennummerierung",
		"infinite_scroll":        "Unendliches Scrollen",
		"previous_page":          "Vorherige",
		"next_page":              "Nächste",
		"first_page":             "Erste",
		"last_page":              "Letzte",
		"showing_records":        "Zeigt",
		"to":                     "bis",
		"load_more":              "Mehr laden",
		"loading_more":           "Weitere Datensätze werden geladen...",
		"all_records_loaded":     "Alle Datensätze geladen",
	},
	"fr": {
		"dashboard":              "Tableau de bord",
		"entities":               "Entités",
		"resources":              "Ressources",
		"account":                "Compte",
		"settings":               "Paramètres & Préférences",
		"orders":                 "Commandes",
		"billing":                "Facturation",
		"security":               "Sécurité",
		"biometrics":             "Biométrie",
		"devices":                "Appareils",
		"telemetry":              "Télémétrie",
		"tickets":                "Tickets",
		"firmware":               "Micrologiciel",
		"operators":              "Équipe",
		"analytics":              "Métriques",
		"theme":                  "Thème",
		"language":               "Langue",
		"sign_out":               "Déconnexion",
		"search":                 "Rechercher",
		"add":                    "Ajouter",
		"save":                   "Enregistrer",
		"cancel":                 "Annuler",
		"delete":                 "Supprimer",
		"edit":                   "Modifier",
		"status":                 "Statut",
		"active_session":         "Session active",
		"dark_mode":              "Mode sombre",
		"light_mode":             "Mode clair",
		"theme_active_dark":      "Mode sombre actif",
		"theme_active_light":     "Mode clair actif",
		"refresh":                "Actualiser",
		"system":                 "Système",
		"control_center":         "Centre de contrôle",
		"sign_in":                "Connexion",
		"sign_in_title":          "Connexion à votre espace",
		"sign_in_subtitle":       "Entrez vos identifiants autorisés pour accéder au panneau d'administration.",
		"username_or_email":      "Nom d'utilisateur ou e-mail",
		"password":               "Mot de passe",
		"remember_me":            "Se souvenir de moi",
		"enter_credentials":      "Veuillez entrer le nom d'utilisateur et le mot de passe",
		"quick_demo_roles":       "RÔLES DE DÉMONSTRATION (CLIQUER POUR REMPLIR)",
		"enclave_secured":        "Enclave sécurisée • GoFilament & GoLaravel Engine",
		"switch_to_arabic":       "🇸🇦 العربية (RTL)",
		"switch_to_english":      "🇺🇸 English (LTR)",
		"search_records":         "Rechercher des enregistrements...",
		"filters":                "Filtres",
		"filter":                 "Filtrer",
		"clear_all":              "Tout effacer",
		"select_all":             "Tout sélectionner",
		"deselect_all":           "Tout désélectionner",
		"bulk_actions":           "Actions groupées",
		"bulk_delete":            "Suppression groupée",
		"export_selected":        "Exporter la sélection",
		"clear_selection":        "Effacer la sélection",
		"no_records_found":       "Aucun enregistrement trouvé",
		"rows_per_page":          "Lignes par page",
		"page":                   "Page",
		"of":                     "sur",
		"actions":                "Actions",
		"create_record":          "Créer un enregistrement",
		"edit_record":            "Modifier l'enregistrement",
		"delete_record":          "Supprimer l'enregistrement",
		"delete_selected":        "Supprimer la sélection",
		"delete_confirm_title":   "Supprimer l'enregistrement ?",
		"delete_records_title":   "Supprimer les enregistrements ?",
		"delete_confirm":         "Êtes-vous sûr de vouloir supprimer cet enregistrement ? Cette action est irréversible.",
		"delete_records_confirm": "Êtes-vous sûr de vouloir supprimer les enregistrements sélectionnés ? Cette action est irréversible.",
		"record_created":         "Enregistrement créé avec succès",
		"record_updated":         "Enregistrement mis à jour avec succès",
		"record_deleted":         "Enregistrement supprimé avec succès",
		"records_deleted":        "Enregistrements supprimés avec succès",
		"operation_failed":       "L'opération a échoué. Veuillez réessayer.",
		"show_relations":         "Afficher les entités associées",
		"hide_relations":         "Masquer les entités associées",
		"toggle_relations":       "Basculer les relations",
		"relations":              "Relations",
		"attributes":             "Attributs",
		"manage":                 "Gérer",
		"related_records":        "Enregistrements associés",
		"back":                   "Retour",
		"operational_dashboard":  "Tableau de bord opérationnel",
		"admin_dashboard":        "Tableau de bord d'administration",
		"performance_overview":   "Aperçu des performances",
		"dashboard_subtitle":     "Métriques opérationnelles en direct, télématique et analyse matérielle",
		"refresh_dashboard":      "Actualiser le tableau de bord",
		"active_entities":        "Entités actives",
		"system_status":          "État du système",
		"operational":            "Opérationnel",
		"multi_tenancy":          "Multi-locataire",
		"quick_access_entities":  "Entités à accès rapide",
		"language_picker":        "Sélectionner la langue et l'orientation",
		"direction":              "Orientation",
		"rtl":                    "De droite à gauche (RTL)",
		"ltr":                    "De gauche à droite (LTR)",
		"english":                "Anglais",
		"arabic":                 "Arabe",
		"close":                  "Fermer",
		"apply":                  "Appliquer",
		"save_changes":           "Enregistrer les modifications",
		"sort_by":                "Trier par",
		"clear_sort":             "Effacer le tri",
		"ascending":              "Croissant",
		"descending":             "Décroissant",
		"pagination":             "Pagination",
		"infinite_scroll":        "Défilement Infini",
		"previous_page":          "Précédent",
		"next_page":              "Suivant",
		"first_page":             "Premier",
		"last_page":              "Dernier",
		"showing_records":        "Affichage",
		"to":                     "à",
		"load_more":              "Charger plus",
		"loading_more":           "Chargement de plus d'enregistrements...",
		"all_records_loaded":     "Tous les enregistrements chargés",
	},
}

var defaultLocaleLabels = map[string]string{
	"en": "English",
	"ar": "العربية",
	"es": "Español",
	"fr": "Français",
	"de": "Deutsch",
	"it": "Italiano",
	"pt": "Português",
	"ru": "Русский",
	"zh": "中文",
	"ja": "日本語",
	"ko": "한국어",
	"tr": "Türkçe",
	"fa": "فارسی",
	"ur": "اردو",
	"he": "עברית",
	"id": "Bahasa Indonesia",
	"nl": "Nederlands",
	"pl": "Polski",
	"sv": "Svenska",
}

var defaultLocaleFlags = map[string]string{
	"en": "🇺🇸",
	"ar": "🇸🇦",
	"es": "🇪🇸",
	"fr": "🇫🇷",
	"de": "🇩🇪",
	"it": "🇮🇹",
	"pt": "🇵🇹",
	"ru": "🇷🇺",
	"zh": "🇨🇳",
	"ja": "🇯🇵",
	"ko": "🇰🇷",
	"tr": "🇹🇷",
	"fa": "🇮🇷",
	"ur": "🇵🇰",
	"he": "🇮🇱",
	"id": "🇮🇩",
	"nl": "🇳🇱",
	"pl": "🇵🇱",
	"sv": "🇸🇪",
}

// Locales registers supported languages for the admin panel.
func (p *Panel) Locales(locales ...string) *Panel {
	p.locales = locales
	if p.defaultLocale == "" && len(locales) > 0 {
		p.defaultLocale = locales[0]
	}
	return p
}

// DefaultLocale sets the fallback language code for the panel.
func (p *Panel) DefaultLocale(loc string) *Panel {
	p.defaultLocale = loc
	return p
}

// LocaleLabels configures custom human-readable display names for locales.
func (p *Panel) LocaleLabels(labels map[string]string) *Panel {
	if p.localeLabels == nil {
		p.localeLabels = make(map[string]string)
	}
	for k, v := range labels {
		p.localeLabels[k] = v
	}
	return p
}

// LocaleFlags configures custom flag emojis or icons for locales.
func (p *Panel) LocaleFlags(flags map[string]string) *Panel {
	if p.localeFlags == nil {
		p.localeFlags = make(map[string]string)
	}
	for k, v := range flags {
		p.localeFlags[k] = v
	}
	return p
}

// Translations registers custom translation dictionaries for a given locale.
// It also registers them into GoLaravel's translation engine for c.Trans() lookups.
func (p *Panel) Translations(locale string, lines map[string]string) *Panel {
	if p.translations == nil {
		p.translations = make(map[string]map[string]string)
	}
	if p.translations[locale] == nil {
		p.translations[locale] = make(map[string]string)
	}
	for k, v := range lines {
		p.translations[locale][k] = v
	}
	translation.Default().AddLines(lines, locale)
	return p
}

// GetTranslations returns the combined translation dictionary for a given locale,
// merging panel-specific translations over built-in admin translations with English fallback.
func (p *Panel) GetTranslations(locale string) map[string]string {
	result := make(map[string]string)

	// 1. Built-in defaults
	if def, ok := defaultAdminTranslations[locale]; ok {
		for k, v := range def {
			result[k] = v
		}
	} else if def, ok := defaultAdminTranslations["en"]; ok {
		for k, v := range def {
			result[k] = v
		}
	}

	// 2. Panel-specific overrides
	if p.translations != nil {
		if custom, ok := p.translations[locale]; ok {
			for k, v := range custom {
				result[k] = v
			}
		}
	}

	return result
}

// GetExactTranslations returns only the translations explicitly registered for the given locale
// (either in built-in defaultAdminTranslations[locale] or p.translations[locale]),
// without falling back to English for missing keys.
func (p *Panel) GetExactTranslations(locale string) map[string]string {
	result := make(map[string]string)

	if def, ok := defaultAdminTranslations[locale]; ok {
		for k, v := range def {
			result[k] = v
		}
	}

	if p.translations != nil {
		if custom, ok := p.translations[locale]; ok {
			for k, v := range custom {
				result[k] = v
			}
		}
	}

	return result
}

// ValidateTranslations verifies that all configured locales on the panel (or default locales if none specified)
// contain all keys in RequiredTranslationKeys. Returns an error detailing all missing keys if any are absent.
func (p *Panel) ValidateTranslations() error {
	locales := p.locales
	if len(locales) == 0 {
		locales = []string{"en", "ar"}
	}

	var errs []string
	for _, loc := range locales {
		dict := p.GetExactTranslations(loc)
		missing := ValidateLocaleTranslations(loc, dict)
		if len(missing) > 0 {
			errs = append(errs, fmt.Sprintf("locale %q is missing %d required translation key(s): %s", loc, len(missing), strings.Join(missing, ", ")))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("gofilament translation validation error:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ValidateLocaleTranslations checks a single translation map against RequiredTranslationKeys.
// It returns a slice of all required keys that are either missing or empty.
func ValidateLocaleTranslations(locale string, dict map[string]string) []string {
	var missing []string
	for _, key := range RequiredTranslationKeys {
		val, exists := dict[key]
		if !exists || strings.TrimSpace(val) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

// GetLocales returns all configured language codes.
func (p *Panel) GetLocales() []string {
	return p.locales
}

// GetDefaultLocale returns the default language code.
func (p *Panel) GetDefaultLocale() string {
	if p.defaultLocale != "" {
		return p.defaultLocale
	}
	if len(p.locales) > 0 {
		return p.locales[0]
	}
	return "en"
}

// GetLocaleLabels returns the map of custom or default display names for configured locales.
func (p *Panel) GetLocaleLabels() map[string]string {
	labels := make(map[string]string)
	locales := p.locales
	if len(locales) == 0 {
		locales = []string{"en", "ar"}
	}
	for _, loc := range locales {
		labels[loc] = p.GetLocaleLabel(loc)
	}
	if p.localeLabels != nil {
		for k, v := range p.localeLabels {
			labels[k] = v
		}
	}
	return labels
}

// GetLocaleFlags returns the map of custom or default flag symbols for configured locales.
func (p *Panel) GetLocaleFlags() map[string]string {
	flags := make(map[string]string)
	locales := p.locales
	if len(locales) == 0 {
		locales = []string{"en", "ar"}
	}
	for _, loc := range locales {
		flags[loc] = p.GetLocaleFlag(loc)
	}
	if p.localeFlags != nil {
		for k, v := range p.localeFlags {
			flags[k] = v
		}
	}
	return flags
}

// GetLocaleLabel returns the display name for a given language code.
func (p *Panel) GetLocaleLabel(locale string) string {
	if p.localeLabels != nil {
		if lbl, ok := p.localeLabels[locale]; ok && lbl != "" {
			return lbl
		}
	}
	if lbl, ok := defaultLocaleLabels[locale]; ok && lbl != "" {
		return lbl
	}
	return strings.ToUpper(locale)
}

// GetLocaleFlag returns the flag emoji/icon for a given language code.
func (p *Panel) GetLocaleFlag(locale string) string {
	if p.localeFlags != nil {
		if f, ok := p.localeFlags[locale]; ok && f != "" {
			return f
		}
	}
	if f, ok := defaultLocaleFlags[locale]; ok && f != "" {
		return f
	}
	return "🌐"
}

// resolveLocale detects the user's preferred locale from query, cookie, context, or defaults.
func (p *Panel) resolveLocale(ctx *Context) {
	if ctx == nil || ctx.Context == nil {
		return
	}

	target := ""
	// 1. URL Query Parameter ?lang= or ?locale=
	if q := ctx.Query("lang"); q != "" {
		target = q
	} else if q := ctx.Query("locale"); q != "" {
		target = q
	}

	// 2. Cookie
	if target == "" {
		if c, err := ctx.Cookie("locale"); err == nil && c != "" {
			target = c
		}
	}

	// 3. Current context locale
	if target == "" {
		target = ctx.GetLocale()
	}

	// 4. Default locale
	if target == "" && p.defaultLocale != "" {
		target = p.defaultLocale
	} else if target == "" && len(p.locales) > 0 {
		target = p.locales[0]
	}

	// Validate against supported locales if configured
	if len(p.locales) > 0 && target != "" {
		supported := false
		for _, loc := range p.locales {
			if strings.EqualFold(loc, target) || strings.HasPrefix(strings.ToLower(target), strings.ToLower(loc)+"-") {
				target = loc
				supported = true
				break
			}
		}
		if !supported {
			if p.defaultLocale != "" {
				target = p.defaultLocale
			} else {
				target = p.locales[0]
			}
		}
	}

	if target != "" {
		ctx.SetLocale(target)
		if ctx.Response.ResponseWriter != nil {
			ctx.Response.Header().Set("Content-Language", target)
		}
	}
}

// renderLocaleSwitcherHTML renders an interactive language dropdown pill for the Topbar.
func (p *Panel) renderLocaleSwitcherHTML(ctx *Context) string {
	if len(p.locales) <= 1 {
		return ""
	}

	currentLocale := ctx.Locale()
	currentLabel := p.GetLocaleLabel(currentLocale)
	currentFlag := p.GetLocaleFlag(currentLocale)
	isRTL := ctx.IsRTL()

	alignStyle := "left: 0;"
	if isRTL {
		alignStyle = "right: 0;"
	}

	var sb strings.Builder
	sb.WriteString(`<div class="filament-locale-switcher" style="position: relative;" x-data="{ open: false }">`)
	sb.WriteString(fmt.Sprintf(`
  <button type="button" @click="open = !open" class="btn btn-secondary btn-sm" style="display: flex; align-items: center; gap: 0.375rem; border-radius: 0.5rem; padding: 0.25rem 0.625rem; font-weight: 500; cursor: pointer; border: 1px solid var(--gray-200); background: var(--gray-50);" title="Change Language">
    <span style="font-size: 1rem; line-height: 1;">%s</span>
    <span style="font-size: 0.8125rem;">%s</span>
    <svg style="width: 0.75rem; height: 0.75rem; color: var(--gray-400); flex-shrink: 0;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
    </svg>
  </button>`, currentFlag, html.EscapeString(currentLabel)))

	sb.WriteString(fmt.Sprintf(`
  <div x-show="open" @click.outside="open = false" style="display: none; position: absolute; %s top: 100%%; margin-top: 0.375rem; min-width: 10rem; background: #fff; border: 1px solid var(--gray-200); border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); z-index: 50; overflow: hidden;">
    <div style="padding: 0.375rem 0.75rem; font-size: 0.6875rem; font-weight: 700; text-transform: uppercase; color: var(--gray-400); border-bottom: 1px solid var(--gray-100); display: flex; justify-content: space-between; align-items: center;">
      <span>Language</span>
      <span>%d</span>
    </div>
    <div style="max-height: 14rem; overflow-y: auto;">`, alignStyle, len(p.locales)))

	for _, loc := range p.locales {
		isActive := loc == currentLocale
		label := p.GetLocaleLabel(loc)
		flag := p.GetLocaleFlag(loc)

		bg := "transparent"
		fontWeight := "normal"
		checkMark := ""
		if isActive {
			bg = "var(--primary-50)"
			fontWeight = "600"
			checkMark = `<span style="color: var(--primary-600); font-weight: 700;">✓</span>`
		}

		sb.WriteString(fmt.Sprintf(`
      <a href="%s/locale/switch?lang=%s" style="display: flex; align-items: center; justify-content: space-between; padding: 0.5rem 0.75rem; font-size: 0.8125rem; color: inherit; text-decoration: none; border-bottom: 1px solid var(--gray-50); background: %s;">
        <div style="display: flex; align-items: center; gap: 0.5rem;">
          <span style="font-size: 1rem; line-height: 1;">%s</span>
          <span style="font-weight: %s;">%s</span>
        </div>
        %s
      </a>`, p.path, loc, bg, flag, fontWeight, html.EscapeString(label), checkMark))
	}

	sb.WriteString(`
    </div>
  </div>
</div>`)

	return sb.String()
}
