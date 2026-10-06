import { Link, useNavigate } from "react-router-dom";
import AuthForm from "../components/auth/AuthForm";
import { useAuth } from "../auth/useAuth";
import { verifyPath } from "../auth/verifyPath";

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();

  async function submit(email, password) {
    const { email: registeredEmail } = await register(email, password);
    navigate(verifyPath(registeredEmail), { state: { reason: "registered" } });
  }

  return (
    <AuthForm
      title="Create an account"
      submitLabel="Create account"
      passwordAutoComplete="new-password"
      passwordHint="Between 8 and 72 characters."
      minPasswordLength={8}
      onSubmit={submit}
      footer={<>Already have an account? <Link to="/login" className="font-semibold text-sky-400 hover:underline">Log in</Link></>}
    />
  );
}
