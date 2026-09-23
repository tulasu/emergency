package main

import "traineebox/internal/tickets/domain/models"

type catalogTypeSeed struct {
	Code   string
	Title  string
	Groups []catalogGroupSeed
}

type catalogGroupSeed struct {
	Code          string
	Title         string
	SelectionMode models.TagSelectionMode
	ParentTagCode string
	Tags          []catalogTagSeed
}

type catalogTagSeed struct {
	Code  string
	Title string
}

type catalogServiceSeed struct {
	Code  string
	Title string
}

func tag(code, title string) catalogTagSeed {
	return catalogTagSeed{Code: code, Title: title}
}

func group(code, title string, mode models.TagSelectionMode, parent string, tags ...catalogTagSeed) catalogGroupSeed {
	return catalogGroupSeed{
		Code: code, Title: title, SelectionMode: mode, ParentTagCode: parent, Tags: tags,
	}
}

func yesNoGroup(code, title string) catalogGroupSeed {
	return group(code, title, models.TagSelectionSingle, "",
		tag(code+"_yes", "Да"),
		tag(code+"_no", "Нет"),
	)
}

func commonEmergencyGroups() []catalogGroupSeed {
	return []catalogGroupSeed{
		group("access", "Доступ", models.TagSelectionMulti, "",
			tag("no_access", "Нет доступа"),
		),
		yesNoGroup("threat_people", "Угроза людям"),
		yesNoGroup("medical_help", "Медицинская помощь"),
		yesNoGroup("evacuation", "Требуется эвакуация"),
	}
}

func commonEmergencyWithOffense() []catalogGroupSeed {
	return append(commonEmergencyGroups(),
		group("offense", "Правонарушение", models.TagSelectionMulti, "",
			tag("has_offense", "Есть правонарушение"),
		),
	)
}

func groups101() []catalogGroupSeed {
	return []catalogGroupSeed{
		group("where", "Где", models.TagSelectionSingle, "",
			tag("where_street", "Улица"),
			tag("where_transport", "Транспорт"),
			tag("where_home", "Дом"),
			tag("where_building", "Здание / объект"),
			tag("where_hazard", "Опасный объект"),
		),
		group("fire_sign_street", "Признак пожара (улица)", models.TagSelectionSingle, "where_street",
			tag("street_flame_smoke", "Открытое пламя / Дым"),
			tag("street_burn_smell", "Запах гари"),
		),
		group("fire_sign_transport", "Признак пожара (транспорт)", models.TagSelectionSingle, "where_transport",
			tag("transport_flame_smoke", "Открытое пламя / Дым"),
			tag("transport_alarm", "Сработала пожарная сигнализация"),
		),
		group("access", "Доступ", models.TagSelectionMulti, "",
			tag("no_access", "Нет доступа"),
		),
		group("street_burning", "Улица (пламя, дым)", models.TagSelectionSingle, "street_flame_smoke",
			tag("burn_trash", "Мусор"),
			tag("burn_grass", "Трава, пух"),
			tag("burn_park", "Парк"),
			tag("burn_forest", "Лес"),
			tag("burn_peat", "Торф"),
			tag("burn_light_mast", "Мачта освещения"),
			tag("burn_contact_support", "Опора контактной сети"),
			tag("burn_powerline", "ЛЭП"),
			tag("burn_wires", "Провода"),
			tag("burn_trees", "Дерево, деревья"),
			tag("burn_person", "Горит человек"),
			tag("burn_unknown", "Что горит неизвестно"),
		),
		group("transport_burning", "Транспорт (пламя, дым)", models.TagSelectionSingle, "transport_flame_smoke",
			tag("tr_public", "Общественный транспорт"),
			tag("tr_car", "Автомашина"),
			tag("tr_accident_fire", "ДТП с пожаром"),
			tag("tr_dangerous_cargo", "Опасный груз"),
			tag("tr_air", "Воздушный транспорт"),
			tag("tr_airport", "Аэропорт"),
			tag("tr_rail", "Ж/Д транспорт"),
			tag("tr_rail_station", "Вокзал ж/д, платформа ж/д"),
			tag("tr_other", "Транспорт прочее"),
			tag("tr_water", "Водный"),
			tag("tr_bridge", "Мост"),
			tag("tr_overpass", "Эстакада"),
			tag("tr_tunnel", "Тоннель"),
			tag("tr_crossing", "Переход подземный/наземный"),
			tag("tr_metro", "Метро"),
			tag("tr_mcc_mcd", "МЦК, МЦД"),
			tag("tr_rail_tracks", "Ж/Д пути"),
			tag("tr_rail_cabinet", "Релейный шкаф Ж/Д"),
		),
		group("scene_place", "Место происшествия", models.TagSelectionSingle, "",
			tag("place_tunnel", "Тоннель"),
			tag("place_pedestrian", "Пешеходный переход"),
		),
		yesNoGroup("threat_people", "Угроза людям"),
		group("offense", "Правонарушение", models.TagSelectionMulti, "",
			tag("has_offense", "Есть правонарушение"),
		),
		yesNoGroup("medical_help", "Медицинская помощь"),
		yesNoGroup("evacuation", "Требуется эвакуация"),
		group("gasification", "Проведена ли газификация", models.TagSelectionSingle, "",
			tag("gas_yes", "Да"),
			tag("gas_no", "Нет"),
			tag("gas_unknown", "Нет данных"),
		),
	}
}

func withGroups(code, title string, groups []catalogGroupSeed) catalogTypeSeed {
	return catalogTypeSeed{Code: code, Title: title, Groups: groups}
}

func emptyType(code, title string) catalogTypeSeed {
	return catalogTypeSeed{Code: code, Title: title}
}

func mergeGroups(parts ...[]catalogGroupSeed) []catalogGroupSeed {
	var out []catalogGroupSeed
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

var seedIncidentTypes = []catalogTypeSeed{
	withGroups("101", "101", groups101()),
	withGroups("102", "102", mergeGroups([]catalogGroupSeed{
		group("what_102", "Что случилось", models.TagSelectionSingle, "",
			tag("crime", "Преступление"),
			tag("public_order", "Нарушение общественного порядка"),
			tag("domestic", "Бытовой конфликт"),
			tag("other_102", "Прочее"),
		),
	}, commonEmergencyWithOffense())),
	withGroups("103", "103", mergeGroups([]catalogGroupSeed{
		group("what_103", "Что случилось", models.TagSelectionSingle, "",
			tag("unconscious", "Без сознания"),
			tag("injury", "Травма"),
			tag("pain", "Боль / недомогание"),
			tag("breathing", "Проблемы с дыханием"),
			tag("other_103", "Прочее"),
		),
		yesNoGroup("threat_people", "Угроза людям"),
		yesNoGroup("medical_help", "Медицинская помощь"),
	})),
	withGroups("104", "104", mergeGroups([]catalogGroupSeed{
		group("what_104", "Что случилось", models.TagSelectionSingle, "",
			tag("gas_smell", "Запах газа"),
			tag("gas_leak", "Утечка газа"),
			tag("no_gas", "Отсутствие газа"),
			tag("other_104", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("accidents_city", "Аварии и происшествия в городском хозяйстве", mergeGroups([]catalogGroupSeed{
		group("utility_kind", "Вид аварии", models.TagSelectionSingle, "",
			tag("heat", "Теплоснабжение"),
			tag("water", "Водоснабжение"),
			tag("power", "Электроснабжение"),
			tag("sewage", "Канализация"),
			tag("other_utility", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("accidents_transport", "Аварии и происшествия на транспортных объектах", mergeGroups([]catalogGroupSeed{
		group("transport_object", "Объект", models.TagSelectionSingle, "",
			tag("metro", "Метро"),
			tag("rail", "Железная дорога"),
			tag("road", "Автодорога"),
			tag("air", "Авиация"),
			tag("water_tr", "Водный транспорт"),
		),
	}, commonEmergencyGroups())),
	withGroups("accidents_hydro", "Аварии на гидротехнических сооружениях", mergeGroups([]catalogGroupSeed{
		group("hydro_kind", "Вид", models.TagSelectionSingle, "",
			tag("dam", "Плотина / дамба"),
			tag("canal", "Канал"),
			tag("pump", "Насосная станция"),
			tag("other_hydro", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("accidents_hazardous", "Аварии на опасных и производственных объектах", mergeGroups([]catalogGroupSeed{
		group("hazard_kind", "Вид", models.TagSelectionSingle, "",
			tag("chem", "Химия"),
			tag("explosion_risk", "Взрывоопасно"),
			tag("fire_risk", "Пожар / задымление"),
			tag("other_hazard", "Прочее"),
		),
	}, commonEmergencyWithOffense())),
	emptyType("gratitude", "Благодарность службам"),
	withGroups("uav", "БПЛА", mergeGroups([]catalogGroupSeed{
		group("uav_kind", "Характер", models.TagSelectionSingle, "",
			tag("uav_sighting", "Обнаружение"),
			tag("uav_crash", "Падение"),
			tag("uav_threat", "Угроза"),
		),
	}, commonEmergencyGroups())),
	withGroups("explosion", "Взрыв", mergeGroups([]catalogGroupSeed{
		group("explosion_place", "Где", models.TagSelectionSingle, "",
			tag("exp_street", "Улица"),
			tag("exp_building", "Здание"),
			tag("exp_transport", "Транспорт"),
			tag("exp_other", "Прочее"),
		),
	}, commonEmergencyWithOffense())),
	emptyType("internal_call", "Внутренний звонок (звонок от работников)"),
	emptyType("foreign_language", "Вызов на иностранном языке"),
	emptyType("additional_call", "Дополнительный звонок от заявителя"),
	withGroups("road_obstacles", "Дорожные помехи", []catalogGroupSeed{
		group("obstacle_kind", "Вид помехи", models.TagSelectionSingle, "",
			tag("obstacle_object", "Посторонний предмет"),
			tag("obstacle_damage", "Повреждение покрытия"),
			tag("obstacle_flood", "Подтопление"),
			tag("obstacle_other", "Прочее"),
		),
	}),
	withGroups("traffic_accident", "ДТП", mergeGroups([]catalogGroupSeed{
		group("dtp_place", "Где", models.TagSelectionSingle, "",
			tag("dtp_road", "Проезжая часть"),
			tag("dtp_yard", "Двор"),
			tag("dtp_parking", "Парковка"),
			tag("dtp_other", "Прочее"),
		),
		group("dtp_injured", "Пострадавшие", models.TagSelectionSingle, "",
			tag("dtp_injured_yes", "Да"),
			tag("dtp_injured_no", "Нет"),
		),
	}, commonEmergencyGroups())),
	emptyType("complaint", "Жалоба на действие или бездействие служб"),
	withGroups("animals", "Животные", []catalogGroupSeed{
		group("animal_kind", "Животное", models.TagSelectionSingle, "",
			tag("dog", "Собака"),
			tag("cat", "Кошка"),
			tag("wild", "Дикое животное"),
			tag("other_animal", "Прочее"),
		),
		yesNoGroup("animal_aggression", "Агрессия"),
		yesNoGroup("threat_people", "Угроза людям"),
	}),
	emptyType("consultation", "Консультация"),
	emptyType("non_target_call", "Нецелевой вызов"),
	withGroups("collapse", "Обрушение", mergeGroups([]catalogGroupSeed{
		group("collapse_place", "Где", models.TagSelectionSingle, "",
			tag("col_building", "Здание"),
			tag("col_structure", "Конструкция"),
			tag("col_other", "Прочее"),
		),
	}, commonEmergencyGroups())),
	emptyType("feedback_112", "Отзыв о работе 112 Москва"),
	emptyType("call_cancel", "Отмена вызова"),
	emptyType("wrong_number", "Ошибочно набран номер"),
	emptyType("shift_handover", "Передача дежурства"),
	emptyType("assist_services", "Помощь службам"),
	withGroups("natural_disaster", "Природная стихия", mergeGroups([]catalogGroupSeed{
		group("nature_kind", "Вид", models.TagSelectionSingle, "",
			tag("storm", "Шторм / ветер"),
			tag("flood", "Паводок / наводнение"),
			tag("ice", "Гололед / снег"),
			tag("other_nature", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("other", "Прочие происшествия", commonEmergencyGroups()),
	withGroups("radiation", "Радиация", mergeGroups([]catalogGroupSeed{
		group("radiation_kind", "Характер", models.TagSelectionSingle, "",
			tag("rad_source", "Источник"),
			tag("rad_leak", "Утечка / выброс"),
			tag("rad_suspected", "Подозрение"),
		),
	}, commonEmergencyGroups())),
	withGroups("broken_thermometer", "Разбитый градусник", []catalogGroupSeed{
		group("mercury_place", "Где", models.TagSelectionSingle, "",
			tag("merc_home", "Квартира / дом"),
			tag("merc_public", "Общественное место"),
			tag("merc_other", "Прочее"),
		),
		yesNoGroup("threat_people", "Угроза людям"),
	}),
	withGroups("child_in_danger", "Ребенок в опасности", mergeGroups([]catalogGroupSeed{
		group("child_situation", "Ситуация", models.TagSelectionSingle, "",
			tag("child_locked", "Закрыт / заперт"),
			tag("child_lost", "Потерялся"),
			tag("child_injury", "Травма"),
			tag("child_other", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("gathering", "Сбор", []catalogGroupSeed{
		group("gathering_kind", "Характер", models.TagSelectionSingle, "",
			tag("crowd", "Скопление людей"),
			tag("rally", "Мероприятие"),
			tag("other_gathering", "Прочее"),
		),
	}),
	withGroups("water_accumulation", "Скопление воды", mergeGroups([]catalogGroupSeed{
		group("water_place", "Где", models.TagSelectionSingle, "",
			tag("water_street", "Улица"),
			tag("water_basement", "Подвал"),
			tag("water_yard", "Двор"),
			tag("water_other", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("fatal_outcome", "Смертельный исход", []catalogGroupSeed{
		yesNoGroup("medical_help", "Медицинская помощь"),
		group("offense", "Правонарушение", models.TagSelectionMulti, "",
			tag("has_offense", "Есть правонарушение"),
		),
	}),
	withGroups("social_assistance", "Социальная помощь", []catalogGroupSeed{
		group("social_kind", "Вид помощи", models.TagSelectionSingle, "",
			tag("social_alone", "Одинокий / беспомощный"),
			tag("social_homeless", "Бездомный"),
			tag("social_other", "Прочее"),
		),
	}),
	emptyType("info_101", "Справка 101"),
	emptyType("info_102", "Справка 102"),
	emptyType("info_103", "Справка 103"),
	emptyType("info_104", "Справка 104"),
	emptyType("info_gibdd", "Справка ГИБДД"),
	emptyType("info_city", "Справка Городское хозяйство"),
	emptyType("info_mchs", "Справка МЧС"),
	emptyType("test_call", "Тестовый вызов"),
	emptyType("technical_failure", "Технический сбой (сбой в работе с оборудованием 112 Москва)"),
	emptyType("training", "Тренировка"),
	withGroups("emergency_notification", "Уведомление о ЧС", []catalogGroupSeed{
		group("chs_kind", "Характер", models.TagSelectionSingle, "",
			tag("chs_info", "Информирование"),
			tag("chs_warning", "Предупреждение"),
			tag("chs_other", "Прочее"),
		),
	}),
	withGroups("threat_explosion", "Угроза взрыва/террористического акта", mergeGroups([]catalogGroupSeed{
		group("threat_exp_place", "Где", models.TagSelectionSingle, "",
			tag("tex_building", "Здание"),
			tag("tex_transport", "Транспорт"),
			tag("tex_street", "Улица"),
			tag("tex_other", "Прочее"),
		),
	}, commonEmergencyWithOffense())),
	withGroups("threat_hazardous_release", "Угроза выброса опасных веществ и радиации", mergeGroups([]catalogGroupSeed{
		group("threat_haz_kind", "Вид угрозы", models.TagSelectionSingle, "",
			tag("thaz_chem", "Химия"),
			tag("thaz_rad", "Радиация"),
			tag("thaz_unknown", "Неизвестно"),
		),
	}, commonEmergencyGroups())),
	withGroups("threat_collapse", "Угроза обрушения", mergeGroups([]catalogGroupSeed{
		group("threat_col_place", "Где", models.TagSelectionSingle, "",
			tag("tcol_building", "Здание"),
			tag("tcol_structure", "Конструкция"),
			tag("tcol_other", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("person_in_danger", "Человек в опасности", mergeGroups([]catalogGroupSeed{
		group("person_situation", "Ситуация", models.TagSelectionSingle, "",
			tag("person_trapped", "Заблокирован"),
			tag("person_fall", "Падение / высота"),
			tag("person_water", "Водоем"),
			tag("person_other", "Прочее"),
		),
	}, commonEmergencyGroups())),
	withGroups("ecological", "Экологическое происшествие", mergeGroups([]catalogGroupSeed{
		group("eco_kind", "Вид", models.TagSelectionSingle, "",
			tag("eco_spill", "Разлив"),
			tag("eco_air", "Загрязнение воздуха"),
			tag("eco_waste", "Отходы"),
			tag("eco_other", "Прочее"),
		),
	}, commonEmergencyGroups())),
}

