package config

type Config struct {
	Name         string
	Addr         string
	FileBasePath string

	MySql struct {
		DataSource string
	}

	Redis struct {
		Addr     string
		Password string
		DB       int
	}

	JWT struct {
		Secret string
		Expire string
	}

	Monitor struct {
		Type      string
		MaxRecord int

		File struct {
			Path     string
			StudTime int
		}
	}
}
