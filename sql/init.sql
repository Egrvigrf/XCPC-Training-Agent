CREATE TABLE `users` (
    `id` int NOT NULL AUTO_INCREMENT,
                         `name` varchar(255) NOT NULL,
                         `phone` varchar(11) NOT NULL,
                         `password` varchar(255) NOT NULL,
                         `status` tinyint NOT NULL,
                         `is_system` tinyint NOT NULL,
                         `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                         `delete_at` timestamp NULL DEFAULT NULL,
                         PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;