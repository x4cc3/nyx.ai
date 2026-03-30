import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

describe("Button", () => {
  it("forwards props when rendered with asChild", () => {
    const onClick = jest.fn();

    render(
      <Button asChild aria-label="Open dashboard" title="Dashboard" onClick={onClick}>
        <a href="/dashboard">Dashboard</a>
      </Button>,
    );

    const link = screen.getByRole("link", { name: "Open dashboard" });
    expect(link).toHaveAttribute("title", "Dashboard");

    fireEvent.click(link);
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});

function TabsHarness() {
  const [value, setValue] = useState("one");

  return (
    <Tabs value={value} onValueChange={setValue}>
      <TabsList>
        <TabsTrigger value="one">One</TabsTrigger>
        <TabsTrigger value="two">Two</TabsTrigger>
      </TabsList>
      <TabsContent value="one">Panel one</TabsContent>
      <TabsContent value="two">Panel two</TabsContent>
    </Tabs>
  );
}

describe("Tabs", () => {
  it("supports keyboard navigation and panel linkage", () => {
    render(<TabsHarness />);

    const firstTab = screen.getByRole("tab", { name: "One" });
    const secondTab = screen.getByRole("tab", { name: "Two" });

    expect(firstTab).toHaveAttribute("aria-selected", "true");
    expect(firstTab).toHaveAttribute("aria-controls");
    expect(screen.getByRole("tabpanel")).toHaveAttribute(
      "aria-labelledby",
      firstTab.getAttribute("id"),
    );

    firstTab.focus();
    fireEvent.keyDown(firstTab, { key: "ArrowRight" });

    expect(secondTab).toHaveFocus();
    expect(secondTab).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Panel two");
  });
});
