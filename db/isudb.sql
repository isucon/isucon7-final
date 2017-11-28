CREATE TABLE `m_item` (
  `item_id` int(11) NOT NULL,
  `power1` bigint(20) NOT NULL,
  `power2` bigint(20) NOT NULL,
  `power3` bigint(20) NOT NULL,
  `power4` bigint(20) NOT NULL,
  `price1` bigint(20) NOT NULL,
  `price2` bigint(20) NOT NULL,
  `price3` bigint(20) NOT NULL,
  `price4` bigint(20) NOT NULL,
  PRIMARY KEY (`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `room_time` (
  `room_name` varchar(191) COLLATE utf8mb4_bin NOT NULL,
  `time` bigint(20) NOT NULL,
  PRIMARY KEY (`room_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `buying` (
  `room_name` varchar(191) COLLATE utf8mb4_bin NOT NULL,
  `item_id` int(11) NOT NULL,
  `ordinal` int(11) NOT NULL,
  `time` bigint(20) NOT NULL,
  PRIMARY KEY (`room_name`,`item_id`,`ordinal`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `adding` (
  `room_name` varchar(191) COLLATE utf8mb4_bin NOT NULL,
  `time` bigint(20) NOT NULL,
  `isu` longtext COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`room_name`,`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

