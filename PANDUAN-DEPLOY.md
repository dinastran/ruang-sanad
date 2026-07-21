# 🚀 Panduan Deploy RuangSanad ke Server (Bahasa Super Simpel)

Panduan ini menuntun kamu **dari nol** sampai aplikasi bisa dibuka di
**https://rs.nurdin.digital**. Ikuti urut dari atas ke bawah, jangan loncat. 🙂

> Server kamu: **Ubuntu, RAM 1 GB, Hardisk 60 GB, sudah ada 2 website aktif.**
> Aplikasi kita cuma **1 file program kecil** yang jalan di **port 8080**, jadi
> tidak mengganggu 2 website yang sudah ada.

---

## 🧠 Gambaran besar (baca dulu 30 detik)

Cara kerjanya begini:

1. **Di laptop kamu** (Mac): kita "masak" aplikasinya jadi 1 file program +
   file tampilan (namanya *build*).
2. File itu **dikirim ke server** lewat internet (SSH).
3. Di server, program dijalankan otomatis oleh **systemd** (biar hidup terus,
   auto-nyala lagi kalau server restart).
4. Program mendengarkan di **port 8080**.
5. **Cloudflare** yang memegang domain `rs.nurdin.digital` akan meneruskan
   pengunjung ke server kamu di **port 8080**.

Enaknya: **server tidak perlu install apa-apa** (tidak perlu Go, Node, dll).
Semua "masak-memasak" terjadi di laptop kamu. Server cuma menjalankan 1 file.
Ini penting karena RAM server cuma 1 GB. 👍

---

## ✅ Bagian 0 — Yang harus sudah ada

**Di laptop kamu (Mac):**
- Sudah bisa menjalankan aplikasi ini secara lokal (Go & Node/npm terpasang).
- Terminal.

**Di server:**
- Kamu punya **IP server** (contoh: `123.45.67.89`).
- Kamu bisa **SSH sebagai root** (atau user yang boleh `sudo`).

**Di Cloudflare:**
- Domain `nurdin.digital` sudah ada di akun Cloudflare kamu. ✔️ (kata kamu sudah)

---

## 🔑 Bagian 1 — Pastikan bisa masuk server tanpa password (SSH key)

Script deploy butuh SSH **tanpa ketik password** (pakai kunci). Cek dulu:

```bash
ssh root@IP_SERVER_KAMU "echo berhasil-masuk"
```

- Kalau muncul `berhasil-masuk` **tanpa diminta password** → ✅ lanjut ke Bagian 2.
- Kalau **diminta password**, pasang kunci dulu (sekali saja):

```bash
# Kalau belum punya kunci SSH, bikin dulu (tekan Enter terus):
ls ~/.ssh/id_ed25519.pub || ssh-keygen -t ed25519

# Kirim kunci ke server (nanti diminta password server 1x terakhir kali):
ssh-copy-id root@IP_SERVER_KAMU

# Tes lagi, harusnya sekarang tidak minta password:
ssh root@IP_SERVER_KAMU "echo berhasil-masuk"
```

> Ganti `IP_SERVER_KAMU` dengan IP server kamu yang asli, di semua perintah.

---

## ⚙️ Bagian 2 — Siapkan file konfigurasi `.deploy`

Di folder project (di laptop), buat file `.deploy` dari contohnya:

```bash
cd /Users/nurdiansyahdisastra/ruang-sanad
cp .deploy.example .deploy
```

Lalu buka `.deploy` (pakai editor apa saja) dan isi seperti ini:

```bash
APP_NAME=rs-nurdin
SERVER_USER=root
SERVER_HOST=IP_SERVER_KAMU
SERVER_PATH=/opt/rs-nurdin
```

Penjelasan bayi:
- `APP_NAME=rs-nurdin` → nama program kita di server (biar beda dari 2 web lain).
- `SERVER_HOST` → IP server kamu.
- `SERVER_PATH=/opt/rs-nurdin` → folder rumah aplikasi di server.

> File `.deploy` **rahasia** dan sudah otomatis di-*ignore* git (tidak akan
> ikut ter-upload ke GitHub). Aman.

---

## 🔍 Bagian 3 — Cek port 8080 masih kosong di server

Biar tidak tabrakan dengan 2 website yang sudah ada:

```bash
ssh root@IP_SERVER_KAMU "ss -tlnp | grep ':8080' || echo 'PORT 8080 KOSONG, aman'"
```

- Kalau muncul **`PORT 8080 KOSONG, aman`** → ✅ lanjut.
- Kalau ada tulisan lain (berarti port 8080 dipakai), kabari saya — nanti kita
  ganti ke port lain (misal 8090) dan sesuaikan.

---

## 🚀 Bagian 4 — Jalankan deploy pertama kali

Ini langkah utamanya. Dari folder project di laptop:

```bash
cd /Users/nurdiansyahdisastra/ruang-sanad
./scripts/deploy.sh
```

Script ini akan otomatis:
1. Build tampilan (frontend) + program (Go) di laptop kamu.
2. Kirim file ke server.
3. Karena belum pernah dideploy, dia jalan **mode first-deploy** dan akan
   **bertanya 2 hal**:

   - **Application Port (default: 8080):** → langsung tekan **Enter** (biar 8080).
   - **Application URL:** → ketik persis:
     ```
     https://rs.nurdin.digital
     ```
     lalu Enter.

4. Script akan membuat file `.env`, memasang systemd service, dan
   **menyalakan aplikasi**. Di akhir muncul:
   ```
   ✓ Service is running
   ═══ FIRST DEPLOY COMPLETE ═══
   ```

🎉 Aplikasi sudah hidup di server (di port 8080), tapi belum bisa diakses dari
internet sampai kita atur firewall + Cloudflare (bagian berikutnya).

> **Kalau nanti mau update aplikasi** (setelah ada perubahan kode), cukup jalankan
> `./scripts/deploy.sh` lagi. Dia otomatis tahu ini "update" dan hanya menimpa
> file + restart. Gampang. (lihat Bagian 8)

---

## 🧱 Bagian 5 — Buka port 8080 di firewall server

Cloudflare perlu bisa "menelepon" server kamu di port 8080. Buka portnya:

```bash
# Cek dulu firewall aktif atau tidak:
ssh root@IP_SERVER_KAMU "ufw status"
```

- Kalau hasilnya **`Status: active`**, buka port 8080:
  ```bash
  ssh root@IP_SERVER_KAMU "ufw allow 8080/tcp && ufw reload"
  ```
- Kalau hasilnya **`Status: inactive`**, port sudah terbuka, **lewati** langkah ini.

Sekarang tes dari laptop, apakah aplikasi kebuka lewat IP langsung:

```bash
curl -I http://IP_SERVER_KAMU:8080/login
```

Kalau muncul `HTTP/1.1 200 OK` → ✅ mantap, aplikasi hidup & port terbuka.
Lanjut ke Cloudflare.

> 🔒 **Lebih aman (opsional):** daripada membuka 8080 ke seluruh dunia, kamu bisa
> membuka 8080 **hanya untuk IP milik Cloudflare**. Lihat "Bagian Bonus A" di bawah.

---

## ☁️ Bagian 6 — Setting Cloudflare (3 langkah)

Buka https://dash.cloudflare.com → klik domain **nurdin.digital**.

### Langkah 6.1 — Arahkan subdomain `rs` ke server (DNS)

1. Menu kiri: **DNS → Records**.
2. Klik **Add record**, isi:
   - **Type:** `A`
   - **Name:** `rs`   *(ini yang bikin jadi rs.nurdin.digital)*
   - **IPv4 address:** `IP_SERVER_KAMU`
   - **Proxy status:** **Proxied** (awan **oranye** 🟠 — WAJIB oranye, jangan abu-abu)
3. **Save**.

### Langkah 6.2 — Suruh Cloudflare pakai port 8080 (Origin Rule)

Ini inti permintaan kamu: arahkan hostname ke port 8080.

1. Menu kiri: **Rules → Origin Rules** → **Create rule**.
2. **Rule name:** `rs ke port 8080`
3. Bagian **When incoming requests match…** (If):
   - Field: **Hostname**
   - Operator: **equals**
   - Value: `rs.nurdin.digital`
4. Bagian **Then… / Rewrite to:**
   - Centang **Destination Port**
   - Isi: **8080**
5. **Deploy** / Save.

Artinya: setiap pengunjung `rs.nurdin.digital`, Cloudflare akan meneruskan ke
server kamu **di port 8080**. ✔️

### Langkah 6.3 — Atur mode SSL khusus subdomain ini (penting!)

Aplikasi kita di port 8080 berbicara **HTTP biasa** (tanpa sertifikat). Jadi
Cloudflare harus "ngobrol HTTP" ke server. Caranya: set mode **Flexible**, tapi
**HANYA untuk rs.nurdin.digital** supaya **2 website lama kamu tidak terganggu**.

1. Menu kiri: **Rules → Configuration Rules** → **Create rule**.
2. **Rule name:** `rs SSL flexible`
3. **When incoming requests match…** (If):
   - Field: **Hostname**, Operator: **equals**, Value: `rs.nurdin.digital`
4. **Then the settings are…**: cari **SSL** → pilih **Flexible**.
5. **Deploy** / Save.

> ❓ *Kenapa tidak ubah SSL global?* Karena mode SSL global berlaku ke SEMUA
> subdomain. Kalau diubah ke Flexible, 2 website lama kamu (yang pakai HTTPS
> beneran) bisa rusak. Pakai **Configuration Rule** = ubah hanya untuk `rs`. Aman. 🛡️
>
> Kalau menu **Configuration Rules** tidak ada di plan kamu, kabari saya — ada
> cara alternatif (pakai Cloudflare Tunnel, lihat Bagian Bonus B, malah lebih aman).

---

## 🎉 Bagian 7 — Tes buka di browser

Tunggu ~1–2 menit biar Cloudflare menyebar, lalu buka:

```
https://rs.nurdin.digital
```

Harusnya muncul halaman **login RuangSanad**. **SELESAI!** 🥳

Kalau belum muncul, sabar 2–3 menit (DNS/Cloudflare kadang butuh waktu), lalu
coba **hard refresh** (Cmd+Shift+R) atau buka di jendela **incognito**.

---

## 🔁 Bagian 8 — Cara UPDATE aplikasi ke depannya

Setiap kali ada perubahan kode dan mau memperbarui yang di server, cukup:

```bash
cd /Users/nurdiansyahdisastra/ruang-sanad
./scripts/deploy.sh
```

Karena aplikasi sudah pernah dideploy, script otomatis masuk **mode update**:
build ulang di laptop → kirim → restart. Data (database santri, dll) **tidak
hilang** karena disimpan di folder `data/` yang tidak ditimpa. ✔️

---

## 🛠️ Bagian 9 — Perintah berguna (copy-paste kalau perlu)

```bash
# Lihat log aplikasi secara live (Ctrl+C untuk keluar):
ssh root@IP_SERVER_KAMU "journalctl -u rs-nurdin -f"

# Cek status aplikasi (hidup/mati):
ssh root@IP_SERVER_KAMU "systemctl status rs-nurdin"

# Restart aplikasi:
ssh root@IP_SERVER_KAMU "systemctl restart rs-nurdin"

# Nyalakan aplikasi (kalau mati):
ssh root@IP_SERVER_KAMU "systemctl start rs-nurdin"

# Matikan aplikasi:
ssh root@IP_SERVER_KAMU "systemctl stop rs-nurdin"
```

---

## 🩹 Bagian 10 — Kalau ada masalah (Troubleshooting)

**A. Buka rs.nurdin.digital muncul error 502 / 521 / "Web server is down".**
- Berarti Cloudflare tidak bisa menjangkau aplikasi di port 8080. Cek:
  1. Aplikasi hidup? `ssh root@IP "systemctl status rs-nurdin"`
  2. Port 8080 terbuka? `ssh root@IP "ufw status"` → pastikan `8080/tcp ALLOW`.
  3. Tes langsung: `curl -I http://IP_SERVER_KAMU:8080/login` → harus `200 OK`.
  4. Origin Rule port 8080 sudah "Deploy"? DNS `rs` sudah **Proxied (oranye)**?

**B. Muncul "redirect loop" / halaman berputar-putar.**
- Biasanya mode SSL. Pastikan Configuration Rule untuk `rs.nurdin.digital` = **Flexible**.

**C. Deploy gagal "Cannot connect to server via SSH".**
- Ulangi **Bagian 1** (SSH key). Pastikan `ssh root@IP "echo ok"` jalan tanpa password.

**D. Aplikasi mati sendiri / kehabisan RAM (server cuma 1 GB).**
- Cek pemakaian: `ssh root@IP "free -h"`.
- Aplikasi ini ringan (~30–80 MB), tapi kalau server sudah mepet karena 2 web
  lain, pertimbangkan tambah **swap** 1 GB:
  ```bash
  ssh root@IP_SERVER_KAMU "fallocate -l 1G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile && echo '/swapfile none swap sw 0 0' >> /etc/fstab"
  ```

**E. Lihat error detail aplikasi:**
```bash
ssh root@IP_SERVER_KAMU "journalctl -u rs-nurdin -n 50 --no-pager"
```

---

## ➕ Bagian 11 — (Opsional) Aktifkan Login Google & Email

Aplikasi tetap jalan tanpa ini. Kalau mau mengaktifkan, edit `.env` di server:

```bash
ssh root@IP_SERVER_KAMU "nano /opt/rs-nurdin/.env"
```

Isi (contoh):
```
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=xxx
GOOGLE_REDIRECT_URL=https://rs.nurdin.digital/auth/google/callback

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=email-kamu@gmail.com
SMTP_PASS=app-password-gmail
FROM_EMAIL=noreply@nurdin.digital
```
Lalu restart: `ssh root@IP_SERVER_KAMU "systemctl restart rs-nurdin"`.

---

## 🎁 Bagian Bonus A — (Opsional, LEBIH AMAN) Kunci port 8080 hanya untuk Cloudflare

Supaya orang tidak bisa mengakses `http://IP_SERVER:8080` langsung (bypass
Cloudflare), izinkan port 8080 **hanya dari IP Cloudflare**:

```bash
ssh root@IP_SERVER_KAMU 'bash -s' <<"EOF"
# Hapus izin 8080 yang terbuka umum (kalau tadi sudah dibuka):
ufw delete allow 8080/tcp 2>/dev/null || true
# Izinkan 8080 hanya dari rentang IP Cloudflare:
for ip in $(curl -s https://www.cloudflare.com/ips-v4); do ufw allow from $ip to any port 8080 proto tcp; done
for ip in $(curl -s https://www.cloudflare.com/ips-v6); do ufw allow from $ip to any port 8080 proto tcp; done
ufw reload
echo "OK: port 8080 sekarang hanya untuk Cloudflare"
EOF
```

---

## 🎁 Bagian Bonus B — (Alternatif paling aman) Cloudflare Tunnel

Kalau kamu **tidak mau membuka port sama sekali** dan tidak mau ribet SSL,
Cloudflare Tunnel adalah cara paling aman & simpel (server "menelepon keluar"
ke Cloudflare, jadi tidak ada port yang dibuka). Kabari saya kalau mau pakai ini —
saya buatkan panduannya. (Untuk sekarang, panduan utama di atas sudah cukup dan
sesuai permintaan kamu: Origin Rule → port 8080.)

---

## 📌 Catatan teknis penting (untuk kamu simpan)

- **Aplikasi dibangun di laptop, bukan di server.** Jadi server tidak butuh Go/
  Node — cocok untuk RAM 1 GB. Yang di-upload cuma: 1 file program + folder
  `dist/` (tampilan) + folder `migrations/` (struktur database).
- **Database** = SQLite, 1 file di `/opt/rs-nurdin/data/app.db`. Migrasi
  (perubahan struktur tabel) **jalan otomatis** setiap aplikasi start. Kamu tidak
  perlu menjalankan migrasi manual.
- **Program jalan sebagai `root`** lewat systemd (paling simpel untuk 1 server
  pribadi). Bisa diperketat nanti kalau perlu.
- **Perubahan yang saya lakukan agar deploy mulus di laptop Mac → server Linux:**
  driver SQLite diganti ke versi **pure-Go (`modernc.org/sqlite`)**, sehingga
  program bisa dibuild jadi 1 file statis untuk Linux **tanpa CGO/compiler C**.
  Program lokal kamu sudah dites dan tetap jalan normal. ✔️

---

Selamat! Kalau ada langkah yang macet, salin pesan errornya ke saya, nanti saya
pandu lagi. 🙏
