import { COURSES } from "../lib/constants";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import { Icons } from "../lib/icons";

const Browse = () => {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex flex-col md:flex-row justify-between items-end gap-4">
        <div>
          <h2 className="text-3xl font-bold text-slate-800">Find your flow</h2>
          <p className="text-slate-500 mt-1">
            Book your spot in our premium classes.
          </p>
        </div>
        <div className="relative w-full md:w-96">
          <div className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
            <Icons.Search />
          </div>
          <input
            type="text"
            placeholder="Search classes..."
            className="w-full pl-10 pr-4 py-2.5 rounded-xl border-none bg-white/80 backdrop-blur-sm shadow-sm focus:ring-2 focus:ring-indigo-500 transition-all placeholder:text-slate-400"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {COURSES.map((course) => (
          <Card
            key={course.id}
            className="p-0 overflow-hidden hover:-translate-y-1 transition-transform duration-300"
          >
            <div className="h-32 bg-linear-to-br from-indigo-100 to-purple-100 flex items-center justify-center text-6xl relative">
              {course.image}
              <div className="absolute top-4 right-4 bg-white/90 backdrop-blur px-3 py-1 rounded-full text-xs font-bold text-indigo-600 shadow-sm">
                {course.type}
              </div>
            </div>
            <div className="p-6">
              <h3 className="text-xl font-bold text-slate-800">
                {course.title}
              </h3>
              <div className="flex items-center gap-4 text-sm text-slate-500 mt-2 mb-6">
                <div className="flex items-center gap-1">
                  <Icons.User /> {course.instructor}
                </div>
                <div className="flex items-center gap-1">
                  <Icons.Clock /> {course.day}, {course.time}
                </div>
              </div>
              <div className="flex items-center justify-between">
                {course.spots === 0 ? (
                  <span className="text-amber-600 text-sm font-semibold flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-amber-500"></span>{" "}
                    Waitlist Only
                  </span>
                ) : (
                  <span
                    className={`text-sm font-semibold flex items-center gap-1 ${
                      course.spots < 5 ? "text-rose-500" : "text-emerald-600"
                    }`}
                  >
                    <span
                      className={`w-2 h-2 rounded-full ${
                        course.spots < 5 ? "bg-rose-500" : "bg-emerald-500"
                      }`}
                    ></span>
                    {course.spots} spots left
                  </span>
                )}
                <Button
                  variant={course.spots === 0 ? "outline" : "primary"}
                  className="text-sm px-6"
                >
                  {course.spots === 0 ? "Join Waitlist" : "Book"}
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
};

export default Browse;
