import React, { useState } from "react";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import { Icons } from "../lib/icons";
import toast from "react-hot-toast";

const Profile = () => {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    username: "john_doe",
    email: "john@example.com",
    firstName: "John",
    lastName: "Doe",
    age: "24",
    height: "175",
    weight: "70",
    bio: "Fitness enthusiast",
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleSave = () => {
    setLoading(true);
    // Mock API call
    setTimeout(() => {
      console.log("Saved profile:", formData);
      setLoading(false);
      toast.success("Profile saved successfully!");
    }, 1000);
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-linear-to-r from-violet-600 to-indigo-600">
            User Profile
          </h1>
          <p className="text-slate-500 mt-2">
            Manage your personal information and preferences
          </p>
        </div>
        <Button
          onClick={handleSave}
          className={loading ? "opacity-50 cursor-not-allowed" : ""}
        >
          {loading ? (
            <span className="flex items-center gap-2">
              <span className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin" />
              Saving...
            </span>
          ) : (
            <>
              <Icons.Check />
              Save Changes
            </>
          )}
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Left Column - Avatar & Basic Info */}
        <div className="space-y-6">
          <Card className="p-6 text-center space-y-4">
            <div className="relative w-32 h-32 mx-auto">
              <img
                src={`https://picsum.photos/id/${1}/200/200`}
                alt="Profile"
                className="w-full h-full rounded-full object-cover border-4 border-indigo-50"
              />
              <button className="absolute bottom-0 right-0 p-2 bg-white rounded-full shadow-lg border border-slate-100 hover:bg-slate-50 transition-colors">
                <Icons.Check />
              </button>
            </div>
            <div>
              <h2 className="text-xl font-bold text-slate-800">
                {formData.firstName} {formData.lastName}
              </h2>
              <p className="text-indigo-600 font-medium">
                @{formData.username}
              </p>
            </div>
          </Card>
        </div>

        {/* Right Column - Form Fields */}
        <div className="md:col-span-2 space-y-6">
          {/* Account Information */}
          <Card className="p-6 space-y-4">
            <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
              <span className="w-1 h-6 bg-indigo-500 rounded-full" />
              Account Information
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Username
                </label>
                <Input
                  name="username"
                  value={formData.username}
                  onChange={handleChange}
                  placeholder="Enter username"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Email
                </label>
                <Input
                  name="email"
                  type="email"
                  value={formData.email}
                  onChange={handleChange}
                  placeholder="Enter email"
                />
              </div>
            </div>
          </Card>

          {/* Personal Details */}
          <Card className="p-6 space-y-4">
            <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
              <span className="w-1 h-6 bg-purple-500 rounded-full" />
              Personal Details
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  First Name
                </label>
                <Input
                  name="firstName"
                  value={formData.firstName}
                  onChange={handleChange}
                  placeholder="First name"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Last Name
                </label>
                <Input
                  name="lastName"
                  value={formData.lastName}
                  onChange={handleChange}
                  placeholder="Last name"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Bio
                </label>
                <Input
                  name="bio"
                  value={formData.bio}
                  onChange={handleChange}
                  placeholder="Short bio"
                />
              </div>
            </div>
          </Card>

          {/* Metrics */}
          <Card className="p-6 space-y-4">
            <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
              <span className="w-1 h-6 bg-pink-500 rounded-full" />
              Body Metrics
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Age
                </label>
                <Input
                  name="age"
                  type="number"
                  value={formData.age}
                  onChange={handleChange}
                  placeholder="Age"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Height (cm)
                </label>
                <Input
                  name="height"
                  type="number"
                  value={formData.height}
                  onChange={handleChange}
                  placeholder="Height"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">
                  Weight (kg)
                </label>
                <Input
                  name="weight"
                  type="number"
                  value={formData.weight}
                  onChange={handleChange}
                  placeholder="Weight"
                />
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Profile;
