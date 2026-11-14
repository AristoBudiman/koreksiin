import os
import openpyxl
from openpyxl.styles import Font, PatternFill, Alignment

OUT_DIR = "/out_samples"
os.makedirs(OUT_DIR, exist_ok=True)


def style_header(ws, cols):
    header_fill = PatternFill(start_color="FF5347", end_color="FF5347", fill_type="solid")
    header_font = Font(bold=True, color="FFFFFF")
    for col_idx, (title, width) in enumerate(cols, start=1):
        cell = ws.cell(row=1, column=col_idx, value=title)
        cell.fill = header_fill
        cell.font = header_font
        cell.alignment = Alignment(vertical="center", wrap_text=True)
        col_letter = openpyxl.utils.get_column_letter(col_idx)
        ws.column_dimensions[col_letter].width = width


# 1. Template Jawaban Siswa Kosong
wb1 = openpyxl.Workbook()
ws1 = wb1.active
ws1.title = "Jawaban Siswa"
style_header(ws1, [("nama_siswa", 28), ("jawaban", 85)])
ws1.append(["Contoh: Andi Pratama", "Tuliskan jawaban esai siswa di kolom ini..."])
wb1.save(os.path.join(OUT_DIR, "template_jawaban_siswa.xlsx"))

# 2. Template Data Latih Kosong
wb2 = openpyxl.Workbook()
ws2 = wb2.active
ws2.title = "Data Latih"
style_header(ws2, [("nama_siswa", 28), ("jawaban", 80), ("nilai", 14)])
ws2.append(["Contoh: Andi Pratama", "Tuliskan jawaban esai siswa di kolom ini...", 85])
wb2.save(os.path.join(OUT_DIR, "template_data_latih.xlsx"))

# 3. File Contoh Fotosintesis 30 Siswa (Lengkap dengan Soal, Rubrik 3 Kriteria, 30 Jawaban + Nilai)
students_data = [
    # --- 10 Jawaban Sangat Lengkap (Semua 3 kriteria terpenuhi) ---
    (
        "Aditya Pratama",
        "Fotosintesis adalah proses pembuatan makanan atau senyawa organik berupa glukosa oleh tumbuhan hijau dengan menggunakan energi cahaya matahari. Proses ini membutuhkan bahan baku utama berupa karbon dioksida dari udara dan air dari dalam tanah, kemudian menghasilkan glukosa serta gas oksigen. Seluruh rangkaian reaksi fotosintesis berlangsung di dalam organel kloroplas pada daun yang memiliki pigmen hijau bernama klorofil untuk menangkap cahaya.",
        96,
    ),
    (
        "Nadia Putri Rahmawati",
        "Proses fotosintesis merupakan penyusunan zat makanan oleh tumbuhan berklorofil dengan memanfaatkan bantuan energi sinar matahari. Dalam proses ini tumbuhan menyerap air melalui akar dan karbon dioksida melalui stomata, lalu mengubahnya menjadi produk akhir berupa glukosa dan oksigen. Reaksi penting ini terjadi di kloroplas daun karena terdapat pigmen klorofil yang berfungsi menyerap energi cahaya.",
        95,
    ),
    (
        "Rizky Ramadhan",
        "Pengertian fotosintesis yaitu reaksi biokimia pembentukan makanan atau karbohidrat oleh tumbuhan dengan bantuan cahaya matahari. Bahan yang diperlukan adalah air (H2O) dan gas karbon dioksida (CO2), sedangkan hasil akhirnya berupa gula glukosa dan gas oksigen (O2). Tempat terjadinya proses fotosintesis berada di dalam kloroplas sel daun yang mengandung zat hijau daun atau klorofil.",
        94,
    ),
    (
        "Siti Nurhaliza",
        "Fotosintesis adalah cara tumbuhan hijau memproduksi makanannya sendiri dengan mengubah energi cahaya matahari menjadi energi kimia. Tumbuhan menggunakan bahan berupa karbon dioksida dan air untuk menghasilkan glukosa sebagai cadangan makanan serta melepaskan oksigen ke udara. Proses tersebut berlangsung di bagian kloroplas pada jaringan mesofil daun dengan bantuan pigmen klorofil.",
        93,
    ),
    (
        "Bagas Setiawan",
        "Fotosintesis merupakan proses pembentukan zat makanan berupa glukosa pada tumbuhan hijau yang dibantu oleh energi cahaya matahari. Bahan dasar yang dipakai dalam reaksi ini adalah air dan karbon dioksida, sementara hasil yang diperoleh adalah glukosa dan oksigen. Proses fotosintesis terjadi di dalam kloroplas karena di dalamnya terdapat klorofil yang menangkap cahaya matahari.",
        95,
    ),
    # Pasangan yang sengaja mirip dengan Bagas Setiawan untuk menguji flag kemiripan (Siswa ke-6)
    (
        "Dimas Arya Putra",
        "Fotosintesis merupakan proses pembentukan zat makanan berupa glukosa pada tumbuhan hijau yang dibantu oleh energi cahaya matahari. Bahan dasar yang digunakan dalam reaksi ini adalah air dan karbon dioksida, sementara hasil yang diperoleh adalah glukosa dan oksigen. Proses fotosintesis terjadi di dalam kloroplas sebab di dalamnya terdapat klorofil yang menangkap cahaya matahari.",
        95,
    ),
    (
        "Aulia Maharani",
        "Fotosintesis didefinisikan sebagai proses tumbuhan hijau memasak makanan sendiri menggunakan energi dari cahaya matahari. Bahan baku yang dibutuhkan tumbuhan meliputi air dan karbon dioksida, yang selanjutnya diubah menjadi hasil berupa glukosa dan oksigen. Kegiatan fotosintesis ini berlangsung di organel kloroplas pada daun berkat adanya pigmen klorofil.",
        92,
    ),
    (
        "Fajar Nugroho",
        "Secara biologi, fotosintesis adalah proses sintesis makanan atau glukosa oleh organisme berklorofil dengan memanfaatkan cahaya matahari. Reaksi ini memerlukan bahan berupa karbon dioksida dan air, serta menghasilkan glukosa dan gas oksigen. Proses ini terjadi di dalam kloroplas daun yang mengandung klorofil untuk menyerap energi cahaya.",
        94,
    ),
    (
        "Intan Permatasari",
        "Fotosintesis ialah proses pembuatan makanan pada tumbuhan hijau dengan bantuan energi cahaya matahari. Dalam reaksi fotosintesis, bahan yang digunakan adalah karbon dioksida dan air, lalu menghasilkan glukosa serta oksigen yang dilepas ke atmosfer. Tempat berlangsungnya reaksi ini adalah di kloroplas daun yang memiliki pigmen klorofil.",
        93,
    ),
    (
        "Kevin Wijaya",
        "Proses fotosintesis adalah pembentukan senyawa glukosa sebagai makanan tumbuhan dengan menggunakan bantuan energi sinar matahari. Tumbuhan membutuhkan bahan baku air dan karbon dioksida, kemudian menghasilkan glukosa dan oksigen. Reaksi tersebut terjadi di organel kloroplas pada daun yang kaya akan pigmen klorofil.",
        92,
    ),

    # --- 10 Jawaban Cukup / Sebagian (Beberapa kriteria atau poin kunci terlewat) ---
    (
        "Larasati Kusuma",
        "Fotosintesis adalah proses pembuatan makanan oleh tumbuhan hijau dengan bantuan cahaya matahari. Bahan yang diperlukan tumbuhan dalam proses ini adalah air dan karbon dioksida untuk menghasilkan glukosa serta oksigen.",
        72,
    ),
    (
        "Muhammad Ilham",
        "Fotosintesis merupakan proses tumbuhan memasak makanannya sendiri menggunakan energi cahaya matahari. Proses ini berlangsung di dalam organel kloroplas pada daun yang mengandung pigmen klorofil. Dari proses ini tumbuhan menghasilkan oksigen.",
        70,
    ),
    (
        "Putri Wulandari",
        "Dalam reaksi fotosintesis, tumbuhan membutuhkan bahan baku berupa air dari tanah dan gas karbon dioksida dari udara, lalu menghasilkan glukosa dan oksigen. Seluruh proses ini terjadi di dalam kloroplas daun yang memiliki pigmen hijau klorofil.",
        74,
    ),
    (
        "Rangga Saputra",
        "Fotosintesis adalah proses pembentukan makanan pada tumbuhan dengan bantuan sinar matahari. Proses ini memakai bahan karbon dioksida dan air. Reaksi ini terjadi di bagian daun tumbuhan yang memiliki klorofil.",
        68,
    ),
    (
        "Dewi Anggraini",
        "Fotosintesis merupakan penyusunan zat makanan oleh tumbuhan hijau dengan memanfaatkan energi cahaya matahari. Tumbuhan menyerap karbon dioksida dan air untuk diubah menjadi glukosa. Kegiatan ini berlangsung di kloroplas daun.",
        75,
    ),
    (
        "Eko Prasetyo",
        "Pengertian fotosintesis adalah proses pembuatan makanan oleh tumbuhan dengan bantuan cahaya matahari. Tumbuhan menyerap air dari akar dan menghasilkan gas oksigen untuk pernapasan makhluk hidup. Proses ini dibantu oleh klorofil di daun.",
        65,
    ),
    (
        "Hana Salsabila",
        "Fotosintesis adalah proses tumbuhan hijau membuat makanan dengan energi sinar matahari. Bahan yang dipakai adalah karbon dioksida dan air, sedangkan tempat terjadinya berada di dalam kloroplas yang mengandung klorofil.",
        76,
    ),
    (
        "Galih Prakoso",
        "Pada proses fotosintesis, bahan berupa air dan karbon dioksida diubah menjadi glukosa serta gas oksigen. Proses pembentukan makanan tumbuhan ini memerlukan energi cahaya matahari agar dapat berjalan.",
        71,
    ),
    (
        "Naufal Hidayat",
        "Fotosintesis adalah proses pembuatan makanan pada tumbuhan hijau menggunakan cahaya matahari. Reaksi ini terjadi di dalam kloroplas daun karena adanya pigmen klorofil, dan bahan utamanya adalah air serta karbon dioksida.",
        76,
    ),
    (
        "Tiara Oktaviani",
        "Tumbuhan melakukan fotosintesis untuk membuat makanan sendiri dengan bantuan cahaya matahari. Dalam proses yang berlangsung di kloroplas daun ini, tumbuhan menghasilkan glukosa sebagai cadangan makanan dan gas oksigen.",
        70,
    ),

    # --- 6 Jawaban Singkat / Kurang Lengkap (Hanya 1 kriteria atau sebagian kecil) ---
    (
        "Vino Bastian",
        "Fotosintesis adalah proses pembuatan makanan oleh tumbuhan hijau dengan menggunakan bantuan energi cahaya matahari pada siang hari.",
        42,
    ),
    (
        "Wulan Guritno",
        "Dalam fotosintesis, bahan yang dibutuhkan adalah karbon dioksida dan air, lalu tumbuhan akan menghasilkan glukosa dan gas oksigen.",
        45,
    ),
    (
        "Yusuf Maulana",
        "Proses fotosintesis terjadi di dalam organel kloroplas pada daun karena di sana terdapat zat hijau daun atau pigmen klorofil.",
        40,
    ),
    (
        "Zahra Khairunnisa",
        "Fotosintesis merupakan cara tumbuhan memasak makanan di daun dengan bantuan sinar matahari sehingga menghasilkan oksigen.",
        48,
    ),
    (
        "Bima Sakti",
        "Tumbuhan membutuhkan air dan sinar matahari untuk melakukan proses fotosintesis di bagian daun yang berwarna hijau.",
        44,
    ),
    (
        "Citra Kirana",
        "Reaksi fotosintesis pada tumbuhan hijau berlangsung di kloroplas dan membutuhkan gas karbon dioksida dari udara sekitar.",
        46,
    ),

    # --- 4 Jawaban Tidak Tepat / Sangat Minim (Kriteria tidak ada atau melenceng) ---
    (
        "Doni Salmanan",
        "Respirasi seluler adalah proses pemecahan glukosa di dalam mitokondria untuk menghasilkan energi ATP bagi makhluk hidup.",
        15,
    ),
    (
        "Ferdiansyah",
        "Tumbuhan menyerap air dan mineral dari dalam tanah menggunakan bulu-bulu akar lalu mengangkutnya melalui jaringan xilem menuju batang.",
        18,
    ),
    (
        "Gita Gutawa",
        "Penguapan air pada tumbuhan disebut transpirasi yang keluar melalui celah stomata pada permukaan bawah daun ketika cuaca panas.",
        15,
    ),
    (
        "Hendra Gunawan",
        "Tumbuhan tumbuh subur apabila diberi pupuk kompos secara rutin dan disiram air setiap pagi dan sore hari di kebun.",
        12,
    ),
]

wb3 = openpyxl.Workbook()

# Sheet 1: Jawaban Siswa (dipakai untuk upload jawaban maupun upload data latih karena memiliki kolom nama_siswa, jawaban, nilai)
ws_sub = wb3.active
ws_sub.title = "Jawaban Siswa"
style_header(ws_sub, [("nama_siswa", 26), ("jawaban", 90), ("nilai", 14)])
for name, ans, score in students_data:
    ws_sub.append([name, ans, score])

# Sheet 2: Info Tugas & Rubrik (Panduan bagi Guru saat membuat tugas contoh)
ws_info = wb3.create_sheet(title="Soal dan Rubrik")
style_header(ws_info, [("Komponen", 28), ("Isi / Keterangan", 95)])
ws_info.append(["Judul Tugas", "Uji Pemahaman Konsep Fotosintesis"])
ws_info.append(["Mata Pelajaran", "Biologi Kelas X"])
ws_info.append([
    "Teks Soal",
    "Jelaskan secara lengkap apa yang dimaksud dengan proses fotosintesis pada tumbuhan hijau, sebutkan bahan baku serta hasil akhirnya, dan jelaskan di mana proses tersebut berlangsung beserta peran pigmen yang terlibat!"
])
ws_info.append(["Nilai Maksimal", 100])
ws_info.append([
    "Kriteria 1 (Bobot 35) - Definisi Proses Fotosintesis",
    "Fotosintesis adalah proses pembuatan atau pembentukan makanan berupa glukosa oleh tumbuhan hijau.\nTumbuhan hijau menggunakan bantuan energi cahaya matahari."
])
ws_info.append([
    "Kriteria 2 (Bobot 35) - Bahan Baku dan Hasil",
    "Tumbuhan membutuhkan bahan baku berupa karbon dioksida (CO2) dan air (H2O).\nReaksi ini menghasilkan glukosa dan gas oksigen (O2)."
])
ws_info.append([
    "Kriteria 3 (Bobot 30) - Tempat Terjadinya dan Peran Klorofil",
    "Reaksi berlangsung di dalam organel kloroplas pada daun.\nDaun memiliki pigmen klorofil untuk menangkap atau menyerap cahaya matahari."
])
ws_info.append([
    "Catatan Uji Kemiripan",
    "Jawaban Bagas Setiawan (baris 6) dan Dimas Arya Putra (baris 7) sengaja dibuat sangat mirip untuk menguji fitur deteksi kemiripan."
])

wb3.save(os.path.join(OUT_DIR, "contoh_fotosintesis_30_siswa.xlsx"))
print("Successfully generated Excel files in", OUT_DIR)

