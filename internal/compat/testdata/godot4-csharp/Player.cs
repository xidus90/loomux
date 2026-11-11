using Godot;

public partial class Player : CharacterBody2D
{
    [Signal]
    public delegate void HitEventHandler();

    public void Jump() => Velocity = new Vector2(Velocity.X, -400);

    public void OnHit() => QueueFree();
}
