import { Link, useNavigate } from "react-router-dom";
import AuthForm from "../components/auth/AuthForm";
import { useAuth } from "../auth/useAuth";
import { verifyPath } from "../auth/verifyPath";

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();

  async function submit(email, password) {
    try {
      await login(email, password);
    } catch (err) {
      // Right password, but the email was never verified
      if (err.status === 403) {
        navigate(verifyPath(email.trim().toLowerCase()), { state: { reason: "unverified" } });
        return;
      }
      throw err;
    }
  }

  return (
    <AuthForm
      title="Log in"
      submitLabel="Log in"
      passwordAutoComplete="current-password"
      onSubmit={submit}
      footer={<>No account yet? <Link to="/register" className="font-semibold text-sky-400 hover:underline">Create one</Link></>}
    />
  );
}
