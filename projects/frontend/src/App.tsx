import { useEffect, useState } from "react";
import "./App.css";

function App() {
  const [name, setName] = useState("");
  const [greeting, setGreeting] = useState<string | null>(null);

  useEffect(() => {
    if (name === "") {
      setGreeting(null);
      return;
    }
    console.log(import.meta.env);

    const fetchingForName = name;
    fetch(import.meta.env.VITE_API + `/greet/${name}`, {
      headers: {
        Accept: "application/json",
      },
    })
      .then((res) => res.json())
      .then((data) => {
        if (name === fetchingForName) {
          setGreeting(data.greeting);
        }
      });
  }, [name]);

  return (
    <div className="App">
      <h1>Greeting App</h1>
      <div className="card">
        <label htmlFor="name">Name</label>{" "}
        <input value={name} onChange={(e) => setName(e.target.value)} />
        <p>{greeting}</p>
      </div>
    </div>
  );
}

export default App;
