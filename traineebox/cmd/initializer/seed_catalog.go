package main

import (
	"context"
	"fmt"

	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"
	ticketsinfra "traineebox/internal/tickets/infrastructure"

	"github.com/spf13/cobra"
)

type catalogTypeSeed struct {
	Code  string
	Title string
	Tags  []catalogTagSeed
}

type catalogTagSeed struct {
	Code  string
	Title string
}

type catalogServiceSeed struct {
	Code  string
	Title string
}

var seedIncidentTypes = []catalogTypeSeed{
	{Code: "101", Title: "101"},
	{Code: "102", Title: "102"},
	{Code: "103", Title: "103"},
	{Code: "104", Title: "104"},
	{Code: "accidents_city", Title: "Аварии и происшествия в городском хозяйстве"},
	{Code: "accidents_transport", Title: "Аварии и происшествия на транспортных объектах"},
	{Code: "accidents_hydro", Title: "Аварии на гидротехнических сооружениях"},
	{Code: "accidents_hazardous", Title: "Аварии на опасных и производственных объектах"},
	{Code: "gratitude", Title: "Благодарность службам"},
	{Code: "uav", Title: "БПЛА"},
	{Code: "explosion", Title: "Взрыв"},
	{Code: "internal_call", Title: "Внутренний звонок (звонок от работников)"},
	{Code: "foreign_language", Title: "Вызов на иностранном языке"},
	{Code: "additional_call", Title: "Дополнительный звонок от заявителя"},
	{Code: "road_obstacles", Title: "Дорожные помехи"},
	{Code: "traffic_accident", Title: "ДТП"},
	{Code: "complaint", Title: "Жалоба на действие или бездействие служб"},
	{Code: "animals", Title: "Животные"},
	{Code: "consultation", Title: "Консультация"},
	{Code: "non_target_call", Title: "Нецелевой вызов"},
	{Code: "collapse", Title: "Обрушение"},
	{Code: "feedback_112", Title: "Отзыв о работе 112 Москва"},
	{Code: "call_cancel", Title: "Отмена вызова"},
	{Code: "wrong_number", Title: "Ошибочно набран номер"},
	{Code: "shift_handover", Title: "Передача дежурства"},
	{Code: "assist_services", Title: "Помощь службам"},
	{Code: "natural_disaster", Title: "Природная стихия"},
	{Code: "other", Title: "Прочие происшествия"},
	{Code: "radiation", Title: "Радиация"},
	{Code: "broken_thermometer", Title: "Разбитый градусник"},
	{Code: "child_in_danger", Title: "Ребенок в опасности"},
	{Code: "gathering", Title: "Сбор"},
	{Code: "water_accumulation", Title: "Скопление воды"},
	{Code: "fatal_outcome", Title: "Смертельный исход"},
	{Code: "social_assistance", Title: "Социальная помощь"},
	{Code: "info_101", Title: "Справка 101"},
	{Code: "info_102", Title: "Справка 102"},
	{Code: "info_103", Title: "Справка 103"},
	{Code: "info_104", Title: "Справка 104"},
	{Code: "info_gibdd", Title: "Справка ГИБДД"},
	{Code: "info_city", Title: "Справка Городское хозяйство"},
	{Code: "info_mchs", Title: "Справка МЧС"},
	{Code: "test_call", Title: "Тестовый вызов"},
	{Code: "technical_failure", Title: "Технический сбой (сбой в работе с оборудованием 112 Москва)"},
	{Code: "training", Title: "Тренировка"},
	{Code: "emergency_notification", Title: "Уведомление о ЧС"},
	{Code: "threat_explosion", Title: "Угроза взрыва/террористического акта"},
	{Code: "threat_hazardous_release", Title: "Угроза выброса опасных веществ и радиации"},
	{Code: "threat_collapse", Title: "Угроза обрушения"},
	{Code: "person_in_danger", Title: "Человек в опасности"},
	{Code: "ecological", Title: "Экологическое происшествие"},
}

var seedServices = []catalogServiceSeed{
	{Code: "sluzhba_101", Title: "Служба 101 (ГУ МЧС России по г.Москве, ГКУ \"Пожарно спасательный центр\" ОДС)"},
	{Code: "fsb", Title: "ФСБ"},
	{Code: "tsemp", Title: "ЦЭМП"},
	{Code: "sluzhba_103", Title: "Служба 103 (ГБУ города Москвы Станция скорой и неотложной медицинской помощи им. А.С. Пучкова)"},
	{Code: "sluzhba_104", Title: "Служба 104 (АО \"МОСГАЗ\" Диспетчерское управление)"},
	{Code: "ogdts", Title: "ОГДЦ"},
	{Code: "tsodd", Title: "ЦОДД (ГКУ \"Центр организации дорожного движения\")"},
	{Code: "gormost", Title: "Гормост (Гормост)"},
	{Code: "mosgortrans", Title: "Мосгортранс"},
	{Code: "avtodorogi", Title: "Автодороги"},
	{Code: "mosvodokanal", Title: "Мосводоканал (АО \"Мосводоканал\")"},
	{Code: "rosseti_mr", Title: "Россети МР"},
	{Code: "moek", Title: "МОЭК"},
	{Code: "dep_zhkh", Title: "Деп. ЖКХ (Департамент ЖКХ)"},
	{Code: "metro", Title: "Метро"},
	{Code: "asu_ns", Title: "АСУ НС (Автоматизированная система управления)"},
	{Code: "mos_bez", Title: "Мос.Без. (Московская Безопасность)"},
	{Code: "112_mos_obl", Title: "112 Мос. обл. (112 Московской области)"},
	{Code: "fgup_rsvo", Title: "ФГУП РСВО (Российские сети вещания и оповещения)"},
	{Code: "oek", Title: "ОЭК (Объединенная энергетическая компания)"},
	{Code: "moslift", Title: "Мослифт (Лифт МСК)"},
	{Code: "mosvodostok", Title: "Мосводосток (Мосводосток)"},
	{Code: "moskollektor", Title: "Москоллектор (Москоллектор)"},
	{Code: "voen_komendatura", Title: "Воен. комендатура (Воен. комендатура)"},
	{Code: "poselenie_chertanovo_severnoe", Title: "Поселение Чертаново Северное (ДДС района Чертаново Северное города Москвы)"},
	{Code: "poselenie_akademicheskiy", Title: "Поселение Академический (ДДС Академического района города Москвы)"},
	{Code: "poselenie_lomonosovskiy", Title: "Поселение Ломоносовский (ДДС Ломоносовского района города Москвы)"},
	{Code: "poselenie_bibirevo", Title: "Поселение Бибирево (ДДС района Бибирево города Москвы)"},
	{Code: "poselenie_yuvao", Title: "Поселение ЮВАО (ДДС префектуры Юго-Восточного административного округа города Москвы)"},
	{Code: "poselenie_vao", Title: "Поселение ВАО (ДДС префектуры Восточного административного округа города Москвы)"},
	{Code: "poselenie_zamoskvoreche", Title: "Поселение Замоскворечье (ДДС района Замоскворечье города Москвы)"},
	{Code: "poselenie_zyablikovo", Title: "Поселение Зябликово (ДДС района Зябликово города Москвы)"},
	{Code: "poselenie_silino", Title: "Поселение Силино (ДДС района Силино города Москвы)"},
	{Code: "oati", Title: "ОАТИ (Объединение Административно-Технических Инспекций города Москвы)"},
	{Code: "gbu_msppn", Title: "ГБУ МСППН (Московская служба психологической помощи населению ГБУ города Москвы)"},
	{Code: "upr_rayona_veshnyaki", Title: "Упр. района Вешняки (ДДС района Вешняки города Москвы)"},
	{Code: "poselenie_kryukovo", Title: "Поселение Крюково (ДДС района Крюково города Москвы)"},
	{Code: "poselenie_matushkino", Title: "Поселение Матушкино (ДДС района Матушкино города Москвы)"},
	{Code: "poselenie_novogireevo", Title: "Поселение Новогиреево (ДДС района Новогиреево города Москвы)"},
	{Code: "poselenie_savelki", Title: "Поселение Савелки (ДДС района Савелки города Москвы)"},
	{Code: "poselenie_staroe_kryukovo", Title: "Поселение Старое Крюково (ДДС района Старое Крюково города Москвы)"},
	{Code: "poselenie_chertanovo_yuzhnoe", Title: "Поселение Чертаново Южное (ДДС района Чертаново Южное города Москвы)"},
	{Code: "poselenie_teplyy_stan", Title: "Поселение Теплый Стан (ДДС района Теплый Стан города Москвы)"},
	{Code: "poselenie_ryazanskiy", Title: "Поселение Рязанский (ДДС Рязанского района города Москвы)"},
	{Code: "poselenie_zelao", Title: "Поселение ЗелАО (ДДС префектуры Зеленоградского административного округа города Москвы)"},
	{Code: "poselenie_pechatniki", Title: "Поселение Печатники (ДДС района Печатники города Москвы)"},
	{Code: "poselenie_sao", Title: "Поселение САО (ДДС префектуры Северного административного округа города Москвы)"},
	{Code: "poselenie_novokosino", Title: "Поселение Новокосино (ДДС района Новокосино города Москвы)"},
	{Code: "poselenie_tsaritsyno", Title: "Поселение Царицыно (ДДС района Царицыно города Москвы)"},
	{Code: "poselenie_marino", Title: "Поселение Марьино (ДДС района Марьино города Москвы)"},
	{Code: "poselenie_kuzminki", Title: "Поселение Кузьминки (ДДС района Кузьминки города Москвы)"},
	{Code: "poselenie_mitino", Title: "Поселение Митино (ДДС района Митино города Москвы)"},
	{Code: "poselenie_savelovskiy", Title: "Поселение Савеловский (ДДС Савеловского района города Москвы)"},
	{Code: "poselenie_schukino", Title: "Поселение Щукино (ДДС района Щукино города Москвы)"},
	{Code: "poselenie_pokrovskoe_streshnevo", Title: "Поселение Покровское-Стрешнево (ДДС района Покровское-Стрешнево города Москвы)"},
	{Code: "poselenie_pervomayskoe", Title: "Поселение Первомайское (ДДС поселения Первомайское в городе Москве)"},
	{Code: "mzhd", Title: "МЖД (Московская железная дорога)"},
	{Code: "112_kal_obl", Title: "112 Кал. обл. (112 Калужская область)"},
	{Code: "poselenie_severnoe_tushino", Title: "Поселение Северное Тушино (ДДС района Северное Тушино города Москвы)"},
	{Code: "poselenie_nagatinskiy_zaton", Title: "Поселение Нагатинский Затон (ДДС района Нагатинский Затон города Москвы)"},
	{Code: "poselenie_dorogomilovo", Title: "Поселение Дорогомилово (ДДС района Дорогомилово города Москвы)"},
	{Code: "poselenie_koptevo", Title: "Поселение Коптево (ДДС района Коптево города Москвы)"},
	{Code: "poselenie_krylatskoe", Title: "Поселение Крылатское (ДДС района Крылатское города Москвы)"},
	{Code: "poselenie_tverskoy", Title: "Поселение Тверской (ДДС Тверского района города Москвы)"},
	{Code: "poselenie_vnukovo", Title: "Поселение Внуково (ДДС района Внуково города Москвы)"},
	{Code: "poselenie_yaroslavskiy", Title: "Поселение Ярославский (ДДС Ярославского района города Москвы)"},
	{Code: "poselenie_yuao", Title: "Поселение ЮАО (ДДС префектуры Южного административного округа города Москвы)"},
	{Code: "poselenie_mihaylovo_yartsevskoe", Title: "Поселение Михайлово-Ярцевское (ДДС поселения Михайлово-Ярцевское города Москвы)"},
	{Code: "poselenie_nekrasovka", Title: "Поселение Некрасовка (ДДС района Некрасовка города Москвы)"},
	{Code: "poselenie_vyhino_zhulebino", Title: "Поселение Выхино-Жулебино (ДДС района Выхино-Жулебино города Москвы)"},
	{Code: "poselenie_moskvoreche_saburovo", Title: "Поселение Москворечье-Сабурово (ДДС района Москворечье-Сабурово города Москвы)"},
	{Code: "poselenie_yuzao", Title: "Поселение ЮЗАО (ДДС префектуры Юго-Западного административного округа города Москвы)"},
	{Code: "poselenie_donskoy", Title: "Поселение Донской (ДДС Донского района города Москвы)"},
	{Code: "poselenie_kurkino", Title: "Поселение Куркино (ДДС района Куркино города Москвы)"},
	{Code: "poselenie_yuzhnoe_butovo", Title: "Поселение Южное Бутово (ДДС района Южное Бутово города Москвы)"},
	{Code: "poselenie_lianozovo", Title: "Поселение Лианозово (ДДС района Лианозово города Москвы)"},
	{Code: "dep_obr", Title: "Деп. Обр. (Московский Департамент Образования)"},
	{Code: "poselenie_lefortovo", Title: "Поселение Лефортово (ДДС района Лефортово города Москвы)"},
	{Code: "poselenie_svao", Title: "Поселение СВАО (ДДС префектуры Северо-Восточного административного округа города Москвы)"},
	{Code: "poselenie_sokolniki", Title: "Поселение Сокольники (ДДС района Сокольники города Москвы)"},
	{Code: "poselenie_izmaylovo", Title: "Поселение Измайлово (ДДС района Измайлово города Москвы)"},
	{Code: "poselenie_bogorodskoe", Title: "Поселение Богородское (ДДС района Богородское города Москвы)"},
	{Code: "poselenie_preobrazhenskoe", Title: "Поселение Преображенское (ДДС района Преображенское города Москвы)"},
	{Code: "poselenie_vostochnoe_izmaylovo", Title: "Поселение Восточное Измайлово (ДДС района Восточное Измайлово города Москвы)"},
	{Code: "poselenie_rostokino", Title: "Поселение Ростокино (ДДС района Ростокино города Москвы)"},
	{Code: "poselenie_sokolinaya_gora", Title: "Поселение Соколиная гора (ДДС района Соколиная гора города Москвы)"},
	{Code: "poselenie_horoshevo_mnevniki", Title: "Поселение Хорошево-Мневники (ДДС района Хорошево-Мневники города Москвы)"},
	{Code: "poselenie_golyanovo", Title: "Поселение Гольяново (ДДС района Гольяново города Москвы)"},
	{Code: "poselenie_severnoe_izmaylovo", Title: "Поселение Северное Измайлово (ДДС района Северное Измайлово города Москвы)"},
	{Code: "poselenie_marfino", Title: "Поселение Марфино (ДДС района Марфино города Москвы)"},
	{Code: "poselenie_metrogorodok", Title: "Поселение Метрогородок (ДДС района Метрогородок города Москвы)"},
	{Code: "poselenie_fili_davydkovo", Title: "Поселение Фили-Давыдково (ДДС района Фили-Давыдково города Москвы)"},
	{Code: "poselenie_yuzhnoe_medvedkovo", Title: "Поселение Южное Медведково (ДДС района Южное Медведково города Москвы)"},
	{Code: "poselenie_ryazanovskoe", Title: "Поселение Рязановское (ДДС поселения Рязановское в городе Москве)"},
	{Code: "dep_prirodopolzovaniya", Title: "Деп. природопользования (Департамент природопользования и охраны окружающей среды города Москвы)"},
	{Code: "poselenie_timiryazevskiy", Title: "Поселение Тимирязевский (ДДС Тимирязевского района города Москвы)"},
	{Code: "poselenie_biryulevo_zapadnoe", Title: "Поселение Бирюлево Западное (ДДС района Бирюлево Западное города Москвы)"},
	{Code: "poselenie_ivanovskiy", Title: "Поселение Ивановский (ДДС Ивановского района города Москвы)"},
	{Code: "poselenie_novo_peredelkino", Title: "Поселение Ново-Переделкино (ДДС района Ново-Переделкино города Москвы)"},
	{Code: "poselenie_vostochnoe_degunino", Title: "Поселение Восточное Дегунино (ДДС района Восточное Дегунино города Москвы)"},
	{Code: "poselenie_hovrino", Title: "Поселение Ховрино (ДДС района Ховрино города Москвы)"},
	{Code: "poselenie_golovinskiy", Title: "Поселение Головинский (ДДС Головинского района города Москвы)"},
	{Code: "poselenie_beskudnikovo", Title: "Поселение Бескудниково (ДДС Бескудниковского района города Москвы)"},
	{Code: "poselenie_levoberezhnyy", Title: "Поселение Левобережный (ДДС района Левобережный города Москвы)"},
	{Code: "poselenie_dmitrovskiy", Title: "Поселение Дмитровский (ДДС Дмитровского района города Москвы)"},
	{Code: "poselenie_sokol", Title: "Поселение Сокол (ДДС района Сокол города Москвы)"},
	{Code: "poselenie_molzhaninovskiy", Title: "Поселение Молжаниновский (ДДС Молжаниновского района города Москвы)"},
	{Code: "poselenie_hamovniki", Title: "Поселение Хамовники (ДДС района Хамовники города Москвы)"},
	{Code: "poselenie_basmannyy", Title: "Поселение Басманный (ДДС Басманного района города Москвы)"},
	{Code: "poselenie_yakimanka", Title: "Поселение Якиманка (ДДС района Якиманка города Москвы)"},
	{Code: "poselenie_meschanskiy", Title: "Поселение Мещанский (ДДС Мещанского района города Москвы)"},
	{Code: "poselenie_presnenskiy", Title: "Поселение Пресненский (ДДС Пресненского района города Москвы)"},
	{Code: "sluzhba_102", Title: "Служба 102 (Дежурная часть ГУ МВД России по г.Москве)"},
	{Code: "poselenie_mosrentgen", Title: "Поселение Мосрентген (ДДС поселения «Мосрентген» в городе Москве)"},
	{Code: "poselenie_yasenevo", Title: "Поселение Ясенево (ДДС района Ясенево города Москвы)"},
	{Code: "mgts", Title: "МГТС (Московская городская телефонная сеть)"},
	{Code: "poselenie_scherbinka", Title: "Поселение Щербинка (ДДС городского округа Щербинка города Москвы)"},
	{Code: "poselenie_kapotnya", Title: "Поселение Капотня (ДДС района Капотня города Москвы)"},
	{Code: "poselenie_troitsk", Title: "Поселение Троицк (ДДС городского округа Троицк города Москвы)"},
	{Code: "poselenie_nagatino_sadovniki", Title: "Поселение Нагатино-Садовники (ДДС района Нагатино-Садовники города Москвы)"},
	{Code: "poselenie_klenovskoe", Title: "Поселение Клёновское (ДДС поселения Клёновское в городе Москве)"},
	{Code: "poselenie_nagornyy", Title: "Поселение Нагорный (ДДС района Нагорный города Москвы)"},
	{Code: "poselenie_babushkinskiy", Title: "Поселение Бабушкинский (ДДС Бабушкинского района города Москвы)"},
	{Code: "poselenie_butyrskiy", Title: "Поселение Бутырский (ДДС Бутырского района города Москвы)"},
	{Code: "poselenie_losinoostrovskiy", Title: "Поселение Лосиноостровский (ДДС Лосиноостровского района города Москвы)"},
	{Code: "poselenie_ostankinskiy", Title: "Поселение Останкинский (ДДС Останкинского района города Москвы)"},
	{Code: "poselenie_otradnoe", Title: "Поселение Отрадное (ДДС района Отрадное города Москвы)"},
	{Code: "poselenie_sviblovo", Title: "Поселение Свиблово (ДДС района Свиблово города Москвы)"},
	{Code: "poselenie_severnoe_medvedkovo", Title: "Поселение Северное Медведково (ДДС района Северное Медведково города Москвы)"},
	{Code: "poselenie_severnyy", Title: "Поселение Северный (ДДС района Северный города Москвы)"},
	{Code: "poselenie_orehovo_borisovo_severnoe", Title: "Поселение Орехово-Борисово Северное (ДДС района Орехово-Борисово Северное города Москвы)"},
	{Code: "poselenie_perovo", Title: "Поселение Перово (ДДС района Перово города Москвы)"},
	{Code: "poselenie_altufevskiy", Title: "Поселение Алтуфьевский (ДДС Алтуфьевского района города Москвы)"},
	{Code: "poselenie_mozhayskiy", Title: "Поселение Можайский (ДДС Можайского района города Москвы)"},
	{Code: "poselenie_ramenki", Title: "Поселение Раменки (ДДС района Раменки города Москвы)"},
	{Code: "poselenie_alekseevskiy", Title: "Поселение Алексеевский (ДДС Алексеевского района города Москвы)"},
	{Code: "poselenie_kuntsevo", Title: "Поселение Кунцево (ДДС района Кунцево города Москвы)"},
	{Code: "poselenie_troparevo_nikulino", Title: "Поселение Тропарево-Никулино (ДДС района Тропарево-Никулино города Москвы)"},
	{Code: "poselenie_vostochnyy", Title: "Поселение Восточный (ДДС района Восточный города Москвы)"},
	{Code: "poselenie_aeroport", Title: "Поселение Аэропорт (ДДС района Аэропорт города Москвы)"},
	{Code: "poselenie_zapadnoe_degunino", Title: "Поселение Западное Дегунино (ДДС района Западное Дегунино города Москвы)"},
	{Code: "poselenie_voykovskiy", Title: "Поселение Войковский (ДДС Войковского района города Москвы)"},
	{Code: "poselenie_horoshevskiy", Title: "Поселение Хорошевский (ДДС Хорошевского района города Москвы)"},
	{Code: "poselenie_begovoy", Title: "Поселение Беговой (ДДС района Беговой города Москвы)"},
	{Code: "poselenie_tekstilschiki", Title: "Поселение Текстильщики (ДДС района Текстильщики города Москвы)"},
	{Code: "poselenie_moskovskiy", Title: "Поселение Московский (ДДС поселения Московский в городе Москве)"},
	{Code: "poselenie_brateevo", Title: "Поселение Братеево (ДДС района Братеево города Москвы)"},
	{Code: "poselenie_filimonkovskoe", Title: "Поселение Филимонковское (ДДС поселения Филимонковское в городе Москве)"},
	{Code: "kanal_im_moskvy", Title: "Канал им. Москвы (ФГБУ \"Канал имени Москвы\")"},
	{Code: "poselenie_lyublino", Title: "Поселение Люблино (ДДС района Люблино города Москвы)"},
	{Code: "poselenie_marushkinskoe", Title: "Поселение Марушкинское (ДДС поселения Марушкинское в городе Москве)"},
	{Code: "dep_truda_i_sots_zaschity", Title: "Деп. труда и соц.защиты (Департамент труда и социальной защиты населения города Москвы)"},
	{Code: "poselenie_orehovo_borisovo_yuzhnoe", Title: "Поселение Орехово-Борисово Южное (ДДС района Орехово-Борисово Южное города Москвы)"},
	{Code: "poselenie_danilovskiy", Title: "Поселение Даниловский (ДДС Даниловского района города Москвы)"},
	{Code: "poselenie_rogovskoe", Title: "Поселение Роговское (ДДС поселения Роговское в городе Москве)"},
	{Code: "poselenie_zao", Title: "Поселение ЗАО (ДДС префектуры Западного административного округа города Москвы)"},
	{Code: "poselenie_nizhegorodskiy", Title: "Поселение Нижегородский (ДДС Нижегородского района города Москвы)"},
	{Code: "poselenie_schapovskoe", Title: "Поселение Щаповское (ДДС поселения Щаповское в городе Москве)"},
	{Code: "poselenie_yuzhnoportovyy", Title: "Поселение Южнопортовый (ДДС Южнопортового района города Москвы)"},
	{Code: "evazhd", Title: "ЭВАЖД (ГБУ \"Учреждение по эксплуатации высотных административных и жилых домов\")"},
	{Code: "poselenie_tsao", Title: "Поселение ЦАО (ДДС префектуры Центрального административного округа города Москвы)"},
	{Code: "poselenie_cheremushki", Title: "Поселение Черемушки (ДДС района Черемушки города Москвы)"},
	{Code: "poselenie_marina_roscha", Title: "Поселение Марьина роща (ДДС района Марьина роща города Москвы)"},
	{Code: "poselenie_krasnoselskiy", Title: "Поселение Красносельский (ДДС Красносельского района города Москвы)"},
	{Code: "poselenie_strogino", Title: "Поселение Строгино (ДДС района Строгино города Москвы)"},
	{Code: "poselenie_szao", Title: "Поселение СЗАО (ДДС префектуры Северо-Западного административного округа города Москвы)"},
	{Code: "poselenie_biryulevo_vostochnoe", Title: "Поселение Бирюлево Восточное (ДДС района Бирюлево Восточное города Москвы)"},
	{Code: "poselenie_chertanovo_tsentralnoe", Title: "Поселение Чертаново Центральное (ДДС района Чертаново Центральное города Москвы)"},
	{Code: "poselenie_gagarinskiy", Title: "Поселение Гагаринский (ДДС Гагаринского района города Москвы)"},
	{Code: "poselenie_obruchevskiy", Title: "Поселение Обручевский (ДДС Обручевского района города Москвы)"},
	{Code: "poselenie_zyuzino", Title: "Поселение Зюзино (ДДС района Зюзино города Москвы)"},
	{Code: "poselenie_konkovo", Title: "Поселение Коньково (ДДС района Коньково города Москвы)"},
	{Code: "poselenie_kotlovka", Title: "Поселение Котловка (ДДС района Котловка города Москвы)"},
	{Code: "poselenie_severnoe_butovo", Title: "Поселение Северное Бутово (ДДС района Северное Бутово города Москвы)"},
	{Code: "poselenie_kosino_uhtomskiy", Title: "Поселение Косино-Ухтомский (ДДС района Косино-Ухтомский города Москвы)"},
	{Code: "poselenie_solntsevo", Title: "Поселение Солнцево (ДДС района Солнцево города Москвы)"},
	{Code: "poselenie_taganskiy", Title: "Поселение Таганский (ДДС района Таганский города Москвы)"},
	{Code: "poselenie_ochakovo_matveevskoe", Title: "Поселение Очаково-Матвеевское (ДДС района Очаково-Матвеевское города Москвы)"},
	{Code: "poselenie_yuzhnoe_tushino", Title: "Поселение Южное Тушино (ДДС района Южное Тушино города Москвы)"},
	{Code: "poselenie_filevskiy_park", Title: "Поселение Филевский парк (ДДС района Филевский парк города Москвы)"},
	{Code: "poselenie_tinao", Title: "Поселение ТИНАО (ДДС префектуры Троицкого и Новомосковского округов города Москвы)"},
	{Code: "poselenie_prospekt_vernadskogo", Title: "Поселение Проспект Вернадского (ДДС района Проспект Вернадского города Москвы)"},
	{Code: "poselenie_desenovskoe", Title: "Поселение Десеновское (ДДС поселения Десеновское в городе Москве)"},
	{Code: "poselenie_arbat", Title: "Поселение Арбат (ДДС района Арбат города Москвы)"},
	{Code: "dezhurno_dispetcherskaya_sluzhba", Title: "Дежурно-диспетчерская служба (Дежурно-диспетчерская служба)"},
	{Code: "poselenie_sosenskoe", Title: "Поселение Сосенское (ДДС поселения Сосенское в городе Москве)"},
	{Code: "poselenie_kokoshkino", Title: "Поселение Кокошкино (ДДС поселения Кокошкино в городе Москве)"},
	{Code: "tsentrregionvodhoz", Title: "Центррегионводхоз (Центррегионводхоз)"},
	{Code: "poselenie_voskresenskoe", Title: "Поселение Воскресенское (ДДС поселения Воскресенское в городе Москве)"},
	{Code: "poselenie_krasnopahorskoe", Title: "Поселение Краснопахорское (ДДС поселения Краснопахорское в городе Москве)"},
	{Code: "poselenie_voronovskoe", Title: "Поселение Вороновское (ДДС поселения Вороновское в городе Москве)"},
	{Code: "mosoblgaz", Title: "Мособлгаз (Мособлгаз)"},
	{Code: "poselenie_vnukovskoe", Title: "Поселение Внуковское (ДДС поселения Внуковское города Москвы)"},
	{Code: "poselenie_novofedorovskoe", Title: "Поселение Новофедоровское (ДДС поселения Новофедоровское города Москвы)"},
	{Code: "poselenie_kievskiy", Title: "Поселение Киевский (ДДС поселения Киевский города Москвы)"},
	{Code: "gbu_ad_vao", Title: "ГБУ АД ВАО (ГБУ Автодороги ВАО)"},
	{Code: "gbu_ad_svao", Title: "ГБУ АД СВАО (ГБУ Автодороги СВАО)"},
	{Code: "gbu_ad_yuvao", Title: "ГБУ АД ЮВАО (ГБУ Автодороги ЮВАО)"},
	{Code: "gbu_ad_zelao", Title: "ГБУ АД ЗелАО (ГБУ Автодороги ЗелАО)"},
	{Code: "gbu_ad_yuzao", Title: "ГБУ АД ЮЗАО (ГБУ Автодороги ЮЗАО)"},
	{Code: "gbu_ad_yuao", Title: "ГБУ АД ЮАО (ГБУ Автодороги ЮАО)"},
	{Code: "gbu_ad_sao", Title: "ГБУ АД САО (ГБУ Автодороги САО)"},
	{Code: "gbu_ad_szao", Title: "ГБУ АД СЗАО (ГБУ Автодороги СЗАО)"},
	{Code: "gbu_ad_tsao", Title: "ГБУ АД ЦАО (ГБУ Автодороги ЦАО)"},
	{Code: "gbu_ad_zao", Title: "ГБУ АД ЗАО (ГБУ Автодороги ЗАО)"},
	{Code: "departament_kultury_goroda_moskvy", Title: "Департамент культуры города Москвы (Департамент культуры города Москвы)"},
	{Code: "gku_tssa", Title: "ГКУ ЦСА (Центр социальной помощи)"},
	{Code: "mosturizm", Title: "Мостуризм (Комитет по туризму города Москвы)"},
	{Code: "departament_stroitelstva", Title: "Департамент строительства (Департамент строительства)"},
	{Code: "komitet_veterinarii", Title: "Комитет ветеринарии (Комитет ветеринарии города Москвы)"},
	{Code: "moszhilinspektsiya", Title: "Мосжилинспекция (Государственная жилищная инспекция города Москвы)"},
}

func newSeedCatalogCmd(cfg config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "seed-catalog",
		Short: "Upsert incident types, tags and emergency services",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := seedCatalog(cmd.Context(), cfg); err != nil {
				return err
			}
			fmt.Println("catalog ready")
			return nil
		},
	}
}

func seedCatalog(ctx context.Context, cfg config.Config) error {
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	catalog := ticketsinfra.NewCatalogRepository(pool)
	for _, t := range seedIncidentTypes {
		it, err := catalog.UpsertIncidentType(ctx, t.Code, t.Title)
		if err != nil {
			return fmt.Errorf("incident type %s: %w", t.Code, err)
		}
		for _, tag := range t.Tags {
			if _, err := catalog.UpsertIncidentTag(ctx, it.ID, tag.Code, tag.Title); err != nil {
				return fmt.Errorf("tag %s/%s: %w", t.Code, tag.Code, err)
			}
		}
	}
	for _, s := range seedServices {
		if _, err := catalog.UpsertService(ctx, s.Code, s.Title); err != nil {
			return fmt.Errorf("service %s: %w", s.Code, err)
		}
	}
	return nil
}
