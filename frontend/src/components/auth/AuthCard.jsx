import Brand from "../ui/Brand";

// Centered page layout shared by the login, register and verify pages
export default function AuthCard({ children }) {
  return (
    <main className="grid min-h-screen place-items-center p-4">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex justify-center"><Brand /></div>
        {children}
      </div>
    </main>
  );
}
