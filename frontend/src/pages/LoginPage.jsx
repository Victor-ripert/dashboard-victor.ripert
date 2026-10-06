import { Link } from "react-router-dom";
import AuthForm from "../components/auth/AuthForm";
import { useAuth } from "../auth/useAuth";

export default function LoginPage() {
  const { login } = useAuth();

  return (
    <AuthForm
      title="Log in"
      submitLabel="Log in"
      passwordAutoComplete="current-password"
      onSubmit={login}
      footer={<>No account yet? <Link to="/register" className="font-semibold text-sky-400 hover:underline">Create one</Link></>}
    />
  );
}
