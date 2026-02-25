# 🚀 ZBEÜ Akıllı Asistan - Hızlı Başlangıç

## 1️⃣ API Key'i Ayarla

PowerShell'de şu komutu çalıştır:

```powershell
$env:GEMINI_API_KEY="AIzaSyDvFzuRr-7a6QzgLps2ItfAdkD7KhR-OhU"
```

## 2️⃣ Uygulamayı Çalıştır

```powershell
cd c:\Users\Cagatay.ok\Desktop\berkhocamladenemeler
go run main.go
```

## 3️⃣ Kullan!

Uygulama başladığında otomatik olarak 3 örnek soru soracak:
- 2026 yaz okulu ne zaman başlıyor?
- 4. sınıfta kaç ders alabilirim?
- AKTS limiti nedir?

Ardından sen de soru sorabilirsin!

## 🧪 Test Modları

```powershell
# Yönetmelik kuralları testi
go run main.go --test-validation

# Web scraping testi
go run main.go --test-scraping

# Gemini API testi
go run main.go --test-gemini

# Yardım
go run main.go --help
```

## ❓ Sorun mu var?

1. **"GEMINI_API_KEY tanımlanmamış" hatası**
   → Adım 1'i tekrar yap

2. **"429 Rate Limit" hatası**
   → Normal! Retry mekanizması çalışıyor, birkaç saniye bekle

3. **Derleme hatası**
   → Go yüklü mü kontrol et: `go version`

## 📚 Daha Fazla Bilgi

- Detaylı kullanım: [README.md](file:///c:/Users/Cagatay.ok/Desktop/berkhocamladenemeler/README.md)
- Proje walkthrough: [walkthrough.md](file:///C:/Users/Cagatay.ok/.gemini/antigravity/brain/7652e8bd-897f-49cc-b5f6-9e6ccfdb9f01/walkthrough.md)

---

**Hazırsın! 🎓**
