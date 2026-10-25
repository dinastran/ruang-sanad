# Label tim dan status jeda CS AI

Di detail lead, pilih **Kelola label** untuk membuat atau mengubah nama dan warna label bersama tim. Label baru tersedia lewat **+ Label** pada semua lead. Menghapus label dari katalog melepaskan label tersebut dari seluruh lead, dengan konfirmasi, tanpa menghapus data lead. Nama dibatasi 60 karakter; duplikat tanpa membedakan kapital ditolak.

CS, Keuangan, dan Super Admin mengikuti akses CRM yang sudah ada. Label yang digunakan aturan follow-up tidak boleh diganti nama atau dihapus sebelum aturan tersebut diubah. Label pilot `uji-followup` tetap dilindungi karena namanya digunakan otomasi; warnanya boleh diubah.

## Perilaku status AI

Daftar dan detail lead menggunakan event jeda/lanjut terakhir berdasarkan waktu kejadian, bukan urutan pengiriman ulang. Jeda dengan `until` RFC3339 yang telah lewat tidak lagi menampilkan badge, meskipun tidak ada event resume. Pembacaan status terpisah dari batas 100 event timeline.

Pada repo `ruangsanadindonesia-art/go-whatsapp-web-multidevice`, handler `integrations/hermes-cs-bridge/admin.go` juga diperbaiki: pengaktifan AI melalui UI `/cs-ai` mengirim `ai.resumed` sesudah jeda lokal berhasil dihapus. Sebelumnya, tindakan UI tersebut tidak memberi tahu dashboard.

## Penerapan

Deploy perubahan dashboard **dan** bridge CS AI. Tidak ada migrasi database baru atau perubahan dependensi aplikasi. Build frontend sebelum backend; jalankan bridge dari modul `integrations/hermes-cs-bridge` sesuai proses deployment yang ada.

Setelah deployment, periksa satu lead yang jedanya berakhir dan satu yang diaktifkan manual melalui `/cs-ai`. Cocokkan tampilan daftar dan detail sesudah memuat ulang. Jangan menghapus seluruh riwayat jeda untuk memperbaiki badge.

Event resume yang hilang sebelum perbaikan tidak otomatis tercipta kembali: jeda lama dengan tenggat yang telah lewat langsung diperhitungkan sebagai selesai; jeda lama yang dibatalkan manual sebelum tenggat memerlukan pengiriman ulang event resume yang benar dari sumber bridge atau menunggu tenggatnya. Jangan menganggap dashboard sebagai status live jika pengiriman event bridge bermasalah. Rekonsiliasi berkala dengan snapshot bridge merupakan pengembangan lanjutan yang disarankan.
