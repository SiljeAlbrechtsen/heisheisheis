#pragma once

#include <string>
#include "Point.h"
#include <filesystem>
#include "SDL_render.h"

namespace TDT4102 {
    enum class FlipImage {
        NONE, HORIZONTAL, VERTICAL
    };

    struct Image {
        // Allow the texture to be read out and drawn despite being private
        friend class AnimationWindow;

    public:
        explicit Image();
        explicit Image(std::filesystem::path pathToImageFile);
        ~Image();

        int width = 0;
        int height = 0;
    private:

        SDL_Texture *texture = nullptr;
        std::filesystem::path src = "non existent file";

        void load(SDL_Renderer* renderer);
        void draw(SDL_Renderer* renderer, TDT4102::Point location, int imageWidth = 0, int imageHeight = 0, double angleDegrees = 0, TDT4102::Point rotationOrigin = {0, 0});
        void drawRegion(SDL_Renderer* renderer, TDT4102::Point location, int imageWidth = 0, int imageHeight = 0, TDT4102::Point sourceTopLeftPoint = {0, 0},  int sourceWidth = 0, int sourceHeight = 0, SDL_RendererFlip flip = SDL_FLIP_NONE);
    };
}
