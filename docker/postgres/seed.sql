CREATE TABLE IF NOT EXISTS weather_records (
    city              VARCHAR(100) PRIMARY KEY,
    temperature_celsius FLOAT       NOT NULL,
    humidity_percent  INT          NOT NULL,
    condition         VARCHAR(50)  NOT NULL,
    wind_speed_kmh    FLOAT        NOT NULL,
    last_updated      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO weather_records (city, temperature_celsius, humidity_percent, condition, wind_speed_kmh) VALUES
    ('jakarta',       32.5,  78, 'partly_cloudy', 14.2),
    ('tokyo',         21.8,  62, 'sunny',          8.5),
    ('new_york',      24.3,  55, 'clear',         12.1),
    ('london',        17.6,  70, 'overcast',       9.8),
    ('paris',         19.4,  65, 'sunny',          7.3),
    ('sydney',        14.2,  58, 'cloudy',        11.4),
    ('dubai',         42.1,  38, 'sunny',         18.7),
    ('singapore',     30.8,  82, 'thunderstorm',  16.3),
    ('mumbai',        28.4,  88, 'rainy',         22.5),
    ('beijing',       26.7,  50, 'hazy',          10.2),
    ('sao_paulo',     19.9,  72, 'partly_cloudy', 13.6),
    ('cairo',         35.8,  28, 'sunny',         20.1),
    ('lagos',         30.2,  84, 'rainy',         17.8),
    ('toronto',       21.5,  60, 'clear',          9.4),
    ('berlin',        18.3,  67, 'partly_cloudy',  8.9),
    ('seoul',         22.6,  58, 'sunny',         11.7),
    ('bangkok',       33.1,  80, 'thunderstorm',  15.9),
    ('mexico_city',   19.2,  55, 'partly_cloudy',  6.8),
    ('istanbul',      21.9,  63, 'sunny',         12.3),
    ('johannesburg',  15.7,  45, 'clear',         14.5)
ON CONFLICT (city) DO NOTHING;
