import { describe, it, expect } from "vitest";
import { extractVideoId, isValidYouTubeUrl } from "./youtubeValidation";

describe("extractVideoId", () => {
  it("extracts from watch URL", () => {
    expect(extractVideoId("https://www.youtube.com/watch?v=dQw4w9WgXcQ")).toBe(
      "dQw4w9WgXcQ"
    );
  });

  it("extracts from short URL", () => {
    expect(extractVideoId("https://youtu.be/dQw4w9WgXcQ")).toBe(
      "dQw4w9WgXcQ"
    );
  });

  it("extracts from embed URL", () => {
    expect(extractVideoId("https://www.youtube.com/embed/dQw4w9WgXcQ")).toBe(
      "dQw4w9WgXcQ"
    );
  });

  it("extracts from shorts URL", () => {
    expect(extractVideoId("https://www.youtube.com/shorts/dQw4w9WgXcQ")).toBe(
      "dQw4w9WgXcQ"
    );
  });

  it("handles URL with extra params", () => {
    expect(
      extractVideoId("https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=30")
    ).toBe("dQw4w9WgXcQ");
  });

  it("handles mobile URL", () => {
    expect(extractVideoId("https://m.youtube.com/watch?v=dQw4w9WgXcQ")).toBe(
      "dQw4w9WgXcQ"
    );
  });

  it("returns null for invalid URL", () => {
    expect(extractVideoId("https://vimeo.com/12345")).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(extractVideoId("")).toBeNull();
  });

  it("returns null for bare video ID", () => {
    expect(extractVideoId("dQw4w9WgXcQ")).toBeNull();
  });
});

describe("isValidYouTubeUrl", () => {
  it("returns true for valid URL", () => {
    expect(isValidYouTubeUrl("https://youtu.be/dQw4w9WgXcQ")).toBe(true);
  });

  it("returns false for invalid URL", () => {
    expect(isValidYouTubeUrl("not-a-url")).toBe(false);
  });
});
