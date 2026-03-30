import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import Drawer from "@/components/ui/Drawer";

describe("Drawer", () => {
  it("traps focus within the drawer and restores focus on close", async () => {
    const onOpenChange = jest.fn();
    const { rerender } = render(
      <>
        <button type="button">Open drawer</button>
        <Drawer open={false} onOpenChange={onOpenChange} title="Menu">
          <button type="button">First action</button>
          <button type="button">Last action</button>
        </Drawer>
      </>,
    );

    const trigger = screen.getByRole("button", { name: "Open drawer" });
    trigger.focus();

    rerender(
      <>
        <button type="button">Open drawer</button>
        <Drawer open onOpenChange={onOpenChange} title="Menu">
          <button type="button">First action</button>
          <button type="button">Last action</button>
        </Drawer>
      </>,
    );

    const dialog = await screen.findByRole("dialog", { name: "Menu" });
    const closeButton = within(dialog).getByRole("button", { name: "Close" });
    const firstAction = screen.getByRole("button", { name: "First action" });
    const lastAction = screen.getByRole("button", { name: "Last action" });

    await waitFor(() => expect(closeButton).toHaveFocus());

    lastAction.focus();
    fireEvent.keyDown(window, { key: "Tab" });
    expect(closeButton).toHaveFocus();

    closeButton.focus();
    fireEvent.keyDown(window, { key: "Tab", shiftKey: true });
    expect(lastAction).toHaveFocus();

    firstAction.focus();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onOpenChange).toHaveBeenCalledWith(false);

    rerender(
      <>
        <button type="button">Open drawer</button>
        <Drawer open={false} onOpenChange={onOpenChange} title="Menu">
          <button type="button">First action</button>
          <button type="button">Last action</button>
        </Drawer>
      </>,
    );

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Open drawer" })).toHaveFocus(),
    );
  });
});
