import { useNavigation } from "react-router";

export function useIsLoading() {
  const navigation = useNavigation();
  return navigation.state === "loading" || navigation.state === "submitting";
}

export function useIsSubmitting() {
  const navigation = useNavigation();
  return navigation.state === "submitting";
}