package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	R2       R2Config
	OpenAI   OpenAIConfig
	Stripe   StripeConfig
	OAuth    OAuthConfig
	Frontend string
}

type AppConfig struct {
	Env    string
	Port   string
	Secret string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string
}

type OpenAIConfig struct {
	APIKey string
}

type StripeConfig struct {
	SecretKey      string
	WebhookSecret  string
	PricePro       string
	PriceAgency    string
}

type OAuthConfig struct {
	Facebook  OAuthProvider
	LinkedIn  OAuthProvider
	Twitter   OAuthProvider
	TikTok    OAuthProvider
	Google    OAuthProvider
}

type OAuthProvider struct {
	ClientID     string
	ClientSecret string
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Ignore missing .env — env vars are sufficient in production
	_ = viper.ReadInConfig()

	accessTTL, _ := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL, _ := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	if refreshTTL == 0 {
		refreshTTL = 7 * 24 * time.Hour
	}

	return &Config{
		App: AppConfig{
			Env:    viper.GetString("APP_ENV"),
			Port:   viper.GetString("APP_PORT"),
			Secret: viper.GetString("APP_SECRET"),
		},
		Database: DatabaseConfig{
			URL: viper.GetString("DATABASE_URL"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		JWT: JWTConfig{
			Secret:     viper.GetString("JWT_SECRET"),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
		},
		R2: R2Config{
			AccountID:       viper.GetString("R2_ACCOUNT_ID"),
			AccessKeyID:     viper.GetString("R2_ACCESS_KEY_ID"),
			SecretAccessKey: viper.GetString("R2_SECRET_ACCESS_KEY"),
			Bucket:          viper.GetString("R2_BUCKET"),
			PublicURL:       viper.GetString("R2_PUBLIC_URL"),
		},
		OpenAI: OpenAIConfig{
			APIKey: viper.GetString("OPENAI_API_KEY"),
		},
		Stripe: StripeConfig{
			SecretKey:     viper.GetString("STRIPE_SECRET_KEY"),
			WebhookSecret: viper.GetString("STRIPE_WEBHOOK_SECRET"),
			PricePro:      viper.GetString("STRIPE_PRICE_PRO"),
			PriceAgency:   viper.GetString("STRIPE_PRICE_AGENCY"),
		},
		OAuth: OAuthConfig{
			Facebook: OAuthProvider{
				ClientID:     viper.GetString("FACEBOOK_APP_ID"),
				ClientSecret: viper.GetString("FACEBOOK_APP_SECRET"),
			},
			LinkedIn: OAuthProvider{
				ClientID:     viper.GetString("LINKEDIN_CLIENT_ID"),
				ClientSecret: viper.GetString("LINKEDIN_CLIENT_SECRET"),
			},
			Twitter: OAuthProvider{
				ClientID:     viper.GetString("TWITTER_CLIENT_ID"),
				ClientSecret: viper.GetString("TWITTER_CLIENT_SECRET"),
			},
			TikTok: OAuthProvider{
				ClientID:     viper.GetString("TIKTOK_CLIENT_KEY"),
				ClientSecret: viper.GetString("TIKTOK_CLIENT_SECRET"),
			},
			Google: OAuthProvider{
				ClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
				ClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
			},
		},
		Frontend: viper.GetString("FRONTEND_URL"),
	}, nil
}
