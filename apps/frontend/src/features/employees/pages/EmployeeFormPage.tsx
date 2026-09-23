import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import type { Employee, OfficeLocation } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Select } from "../../../components/ui/Select";
import { ArrowLeft, UserPlus, Save } from "lucide-react";

export function EmployeeFormPage() {
  const { t } = useTranslation(["employee", "common"]);
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const isEdit = Boolean(id);

  const [nip, setNip] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [department, setDepartment] = useState("");
  const [position, setPosition] = useState("");
  const [officeLocationId, setOfficeLocationId] = useState("");
  const [isActive, setIsActive] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Fetch office locations for select dropdown
  const { data: locations } = useQuery<OfficeLocation[]>({
    queryKey: ["office-locations", "list-all"],
    queryFn: () => api.get<OfficeLocation[]>("/api/v1/office-locations"),
  });

  // Fetch existing employee data if edit mode
  const { data: existingEmployee, isLoading: isEmployeeLoading } = useQuery<Employee>({
    queryKey: ["employees", "detail", id],
    queryFn: () => api.get<Employee>(`/api/v1/employees/${id}`),
    enabled: isEdit,
  });

  useEffect(() => {
    if (existingEmployee) {
      setNip(existingEmployee.nip || existingEmployee.employee_number || "");
      setName(existingEmployee.name || existingEmployee.full_name || "");
      setEmail(existingEmployee.email || "");
      setPhone(existingEmployee.phone || "");
      setDepartment(existingEmployee.department || "");
      setPosition(existingEmployee.position || "");
      setOfficeLocationId(existingEmployee.office_location_id || "");
      setIsActive(existingEmployee.is_active);
    }
  }, [existingEmployee]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    const payload = {
      nip: nip.trim(),
      name: name.trim(),
      email: email.trim() || undefined,
      phone: phone.trim() || undefined,
      department: department.trim() || undefined,
      position: position.trim() || undefined,
      office_location_id: officeLocationId || undefined,
      is_active: isActive,
    };

    try {
      if (isEdit) {
        await api.put(`/api/v1/employees/${id}`, payload);
        toast.show({
          type: "success",
          title: "Berhasil Diperbarui",
          message: `Data karyawan ${name} berhasil disimpan.`,
        });
      } else {
        await api.post("/api/v1/employees", payload);
        toast.show({
          type: "success",
          title: "Karyawan Ditambahkan",
          message: `Karyawan ${name} berhasil didaftarkan.`,
        });
      }
      navigate("/employees");
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Menyimpan Data",
        message: "Periksa kembali kelengkapan form dan keunikan NIP/Email.",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isEdit && isEmployeeLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-2xl mx-auto">
      <div className="flex items-center gap-3">
        <Link to="/employees">
          <Button variant="outline" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> {t("actions.back", { ns: "common" })}
          </Button>
        </Link>
        <div>
          <h1 className="text-xl font-bold text-gray-900">
            {isEdit ? t("form.editTitle", { ns: "employee" }) : t("form.createTitle", { ns: "employee" })}
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            {t("form.personalInfo", { ns: "employee" })}
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <UserPlus className="w-4 h-4 text-indigo-600" />
            <CardTitle>Formulir Karyawan</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">
                  NIP (Nomor Induk Pegawai) *
                </label>
                <Input
                  required
                  placeholder="Contoh: 199001012022031001"
                  value={nip}
                  onChange={(e) => setNip(e.target.value)}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Nama Lengkap *</label>
                <Input
                  required
                  placeholder="Contoh: Budi Pratama"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Email</label>
                <Input
                  type="email"
                  placeholder="budi@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Nomor Telepon</label>
                <Input placeholder="08123456789" value={phone} onChange={(e) => setPhone(e.target.value)} />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Departemen</label>
                <Input
                  placeholder="Contoh: Engineering / HRD"
                  value={department}
                  onChange={(e) => setDepartment(e.target.value)}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Jabatan</label>
                <Input
                  placeholder="Contoh: Staff / Manager"
                  value={position}
                  onChange={(e) => setPosition(e.target.value)}
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Lokasi Kantor Utama</label>
              <Select value={officeLocationId} onChange={(e) => setOfficeLocationId(e.target.value)}>
                <option value="">-- Belum Ditentukan / Bebas --</option>
                {locations?.map((loc) => (
                  <option key={loc.id} value={loc.id}>
                    {loc.name} ({loc.radius_meters}m)
                  </option>
                ))}
              </Select>
            </div>

            <div className="pt-2 flex items-center gap-2">
              <input
                type="checkbox"
                id="is_active_checkbox"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                className="rounded text-indigo-600 focus:ring-indigo-500 h-4 w-4"
              />
              <label
                htmlFor="is_active_checkbox"
                className="text-xs font-medium text-gray-700 cursor-pointer"
              >
                Status Karyawan Aktif (Bisa melakukan absensi)
              </label>
            </div>

            <div className="pt-4 border-t border-gray-100 flex justify-end gap-2">
              <Link to="/employees">
                <Button variant="outline" size="sm" type="button">
                  Batal
                </Button>
              </Link>
              <Button variant="primary" size="sm" type="submit" isLoading={isSubmitting}>
                <Save className="w-4 h-4 mr-1.5" />
                {isEdit ? "Simpan Perubahan" : "Daftarkan Karyawan"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
