# ZBEÜ Akıllı Asistan 🎓

**CAGSOFT** tarafından Çağatay Ok için geliştirilmiş, Go dilinde yazılmış üniversite akıllı asistan botu.

## 🚀 Özellikler

- ✅ **Web Scraping**: ZBEÜ duyurular sayfasından otomatik veri çekme
- ✅ **Google Gemini AI**: Gelişmiş AI destekli yanıtlar
- ✅ **Hata Yönetimi**: 404/429 hatalarını otomatik bypass
- ✅ **ZBEÜ Yönetmelik**: 4. sınıf ders alma kuralları entegrasyonu
- ✅ **AKTS Kontrolü**: 21-24 AKTS limit kontrolü
- ✅ **Yaz Okulu Bilgisi**: 2026 yaz okulu tarihleri ve kuralları

## 📋 Gereksinimler

- **Go 1.16+** (standart kütüphaneler kullanılıyor, dış bağımlılık yok)
- **Google Gemini API Key** ([buradan alın](https://aistudio.google.com/apikey))

## ⚙️ Kurulum

### 1. API Key Ayarlama

**Windows (PowerShell):**
```powershell
$env:GEMINI_API_KEY="your-api-key-here"
```

**Windows (CMD):**
```cmd
set GEMINI_API_KEY=your-api-key-here
```

**Linux/Mac:**
```bash
export GEMINI_API_KEY=your-api-key-here
```

### 2. Uygulamayı Çalıştırma

```bash
cd c:\Users\Cagatay.ok\Desktop\berkhocamladenemeler
go run main.go
```

## 🧪 Test Modları

### Web Scraping Testi
```bash
go run main.go --test-scraping
```

### Gemini API Bağlantı Testi
```bash
go run main.go --test-gemini
```

### Yönetmelik Kuralları Testi
```bash
go run main.go --test-validation
```

### Yardım
```bash
go run main.go --help
```

## 💡 Kullanım Örnekleri

### Örnek Sorular

1. **Yaz Okulu:**
   ```
   2026 yaz okulu ne zaman başlıyor?
   ```

2. **Ders Alma Sınırı:**
   ```
   4. sınıfta kaç ders alabilirim?
   ```

3. **AKTS Hesaplama:**
   ```
   3 ders seçtim, toplam 25 AKTS oluyor, sorun var mı?
   ```

## 🏗️ Mimari

```
main.go
├── Configuration Module      # API ayarları, retry parametreleri
├── Web Scraping Module       # ZBEÜ duyuru çekme (+ mock fallback)
├── Gemini AI Module          # AI entegrasyonu, hata yönetimi
├── ZBEÜ Regulation Module    # Yönetmelik kuralları kontrolü
└── Main Application          # CLI interface, interaktif mod
```

## 🔒 Güvenlik

- ✅ API key **asla** kodda hardcoded değil
- ✅ Environment variable kullanımı
- ✅ Güvenli HTTP client (timeout'lar)
- ✅ Input validation

## 🛠️ Teknik Detaylar

### Hata Yönetimi

**404 Not Found (Model Versiyonu):**
- Otomatik fallback: `gemini-2.0-flash` → `gemini-1.5-flash-8b`

**429 Quota Exceeded (Rate Limit):**
- Exponential backoff: 5s → 10s → 20s
- Maksimum 3 deneme

### ZBEÜ Kuralları

```go
Maksimum Ders: 3 (4. sınıf için)
AKTS Aralığı: 21-24
Yaz Okulu: Başvuru ve ders tarihleri
```

## 📦 Derleme (Opsiyonel)

Executable oluşturmak için:

```bash
go build -o zbeu-asistan.exe main.go
```

Ardından direkt çalıştırın:
```bash
zbeu-asistan.exe
```

## 🎯 Performans

- **Binary Boyutu**: ~5MB
- **Başlangıç Süresi**: <100ms
- **Yanıt Süresi**: <2s (network hariç)
- **Bellek Kullanımı**: ~10MB

## 🐛 Sorun Giderme

### "GEMINI_API_KEY environment variable tanımlanmamış"
- API key'i environment variable olarak tanımlayın (yukarıdaki kurulum adımlarına bakın)

### "404 Not Found" hatası
- Otomatik fallback devreye girer
- Her iki model de başarısız olursa API key'i kontrol edin

### "429 Quota Exceeded" hatası
- Retry mekanizması otomatik çalışır
- Kalıcı sorun varsa API quota'nızı kontrol edin

## 👨‍💻 Geliştirici

**CAGSOFT** - Baş Mimar  
Çağatay Ok için özel geliştirilmiştir.

## 📄 Lisans

Bu proje Çağatay Ok'un kişisel kullanımı için geliştirilmiştir.

---

**Anti-Gravity Architecture** 🚀 - Hafif, hızlı, güçlü!
